package stream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
		CallbackStream: StreamName,     // ✅ 正确：指向客户端自己的 Stream
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
		return resp, nil
	case <-time.After(time.Duration(Deadline-time.Now().UnixMilli()) * time.Millisecond):
		return nil, errors.New("request timeout")
	}
}

// Response 发送响应消息
func Response(originalMsg *StreamMessage, result map[string]interface{}, errCode int, errMsg string) error {
	resp := &StreamMessage{
		MessageID:      generateID(),
		ReplyID:        originalMsg.MessageID,
		ServiceName:    "response",
		CallbackStream: originalMsg.CallbackStream,
		SourceStream:   originalMsg.SourceStream,
		SourceService:  originalMsg.SourceService,
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