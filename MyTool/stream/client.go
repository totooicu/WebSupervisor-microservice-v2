package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"log"
	myredis "github.com/totooicu/go-mytool/redis"
	"github.com/totooicu/go-mytool/encryption"
	"github.com/go-redis/redis/v8"
)

var (
	redisClient *redis.Client
	ctx         = context.Background()
)

func InitClient() error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	_, err := redisClient.Ping(ctx).Result()
	return err
}
func AckAndDelete(ctx context.Context, StreamName , ConsumerGroup, msgID string) error {
	if msgID == "" {
		return errors.New("msgID is empty")
	}
	err := redisClient.XAck(ctx, StreamName, ConsumerGroup, msgID).Err()
	if err != nil {
		return err
	}
	err = redisClient.XDel(ctx, StreamName, msgID).Err()
	if err != nil {
		return err
	}
	return nil
}

// Send 发送请求并等待响应
// timeout_ms_int64 为超时时间，单位毫秒
// 0 表示使用默认超时时间, -1 表示不设置超时时间
func Send(targetStream, targetService string, payload map[string]interface{}, timeout_ms_int64 int64) (*StreamMessage, error) {
	
	var Deadline int64
	{	
		if timeout_ms_int64 == 0 { 
			timeout_ms_int64 = int64(cfg.Stream.GetTimeoutMs)
		}
		Deadline = time.Now().UnixMilli() + timeout_ms_int64
		if timeout_ms_int64 < 0 {
			Deadline = 0
		}
	}

	msgID := generateID()
	respChan := make(chan *StreamMessage, 1)
	mutex.Lock()
	RESPONSES[msgID] = respChan
	mutex.Unlock()

	defer func() {
		mutex.Lock()
		delete(RESPONSES, msgID)
		mutex.Unlock()
	}()

	msg := &StreamMessage{
		MessageID:      msgID,
		ServiceName:    targetService,
		CallbackStream: StreamName,     // 
		SourceStream:   StreamName,
		SourceService:  ServiceName,
		Playload:       payload,
		Sharding:       ShardingInfo{Total: 0},
		Deadline:       Deadline,
	}

	data, _ := json.Marshal(msg)
	if len(data) > MaxMsgSize {
		 if err := sendSharded(msg, data, targetStream); err != nil {
			return nil, err
		}
	} else {
		if err := redisClient.XAdd(ctx, &redis.XAddArgs{
			Stream: targetStream,
			Values: msg.ToMap(),
		}).Err(); err != nil {
			return nil, err
		}
	}
		
	select {
	case resp := <-respChan:
		if resp.ErrCode != 0 {
			return resp,fmt.Errorf("%v", resp.ErrMsg)
		}
		return resp, nil
	case <-time.After(time.Duration(Deadline-time.Now().UnixMilli()) * time.Millisecond):
		return nil, errors.New("request timeout")
	}
}
func SendPing(StreamName string)  (*StreamMessage, error) {
	return Send(StreamName, "ping", nil, 0)
}

func ResponseErr(originalMsg *StreamMessage, errMsg string)error{
	return Response(originalMsg,nil,1,errMsg)
}
func ResponseSucc(originalMsg *StreamMessage, result map[string]interface{})error{
	return Response(originalMsg,result,0,"")
}

// Response 发送响应消息
func Response(originalMsg *StreamMessage, result map[string]interface{}, errCode int, errMsg string) error {
	resp := &StreamMessage{
		MessageID:      generateID(),
		ReplyID:        originalMsg.MessageID,
		ServiceName:    "response",
		CallbackStream: originalMsg.CallbackStream,
		SourceStream:   StreamName,
		SourceService:  originalMsg.ServiceName,
		Playload:       result,
		Sharding:       ShardingInfo{Total: 0},
		ErrCode:        errCode,
		ErrMsg:         errMsg,
		TraceID:        originalMsg.TraceID,
	}
	data, _ := json.Marshal(resp)
	if len(data) > MaxMsgSize {
		return sendSharded(resp, data, originalMsg.CallbackStream)
	}
	return redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: originalMsg.CallbackStream,
		Values: resp.ToMap(),
	}).Err()
}

func generateID() string {
	id, _ := redisClient.Incr(ctx, "stream:msg:id").Result()
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), id)
}

func ReadOne(ctx context.Context, consumerName string) (*redis.XMessage, error) {
	streams, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: consumerName,
		Streams:  []string{StreamName, ">"},
		Count:    1,
		Block:    0,
	}).Result()

	if err != nil {
		log.Println("XReadGroup error:", err)
		return nil, err
	}

	var xmsg redis.XMessage
		for _, stream := range streams {
		for _, message := range stream.Messages {
			xmsg=message
		}}
	return &xmsg, nil
}

func GetMyRedisClient() *myredis.Client {
	return myredis.GetClient(redisClient,ctx)
}
func CacheSet(key string, value interface{}, expiration_ms_int64 time.Duration)( string,error) {
	nkey:=fmt.Sprintf("%s%s:%v",CacheKeyPrefix,key,encryption.HashMD5(fmt.Sprintf("%v",value)))
	return nkey,GetMyRedisClient().SetKey( nkey, value, expiration_ms_int64)
}
func CacheGet(key string, dest interface{}) error {
	return GetMyRedisClient().GetKey(key, dest)
}
func CacheDelete(key string) error {
	return GetMyRedisClient().DeleteKey(key)
}