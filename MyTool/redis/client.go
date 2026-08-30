package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// XMessage 导出Redis XMessage类型
type XMessage = redis.XMessage

type Client struct {
	client *redis.Client
	ctx    context.Context
}

func NewClient(host string, port int, password string, db int) *Client {
	addr:= fmt.Sprintf("%s:%d", host, port)
	if port==-1{
		addr=host
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	return &Client{
		client: client,
		ctx:    ctx,
	}
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) PublishMessage(stream string, message interface{}) error {
	return c.PublishMessageWithMaxLen(stream, message, 1000) // 默认最大消息数量为1000
}

// PublishMessageWithMaxLen 带最大消息数量限制的消息发布
func (c *Client) PublishMessageWithMaxLen(stream string, message interface{}, maxLen int64) error {
	// 检查是否为StreamMessage类型
	if streamMsg, ok := message.(map[string]interface{}); ok {
		// 创建新的map来处理嵌套字段
		msgMap := make(map[string]interface{})

		for key, value := range streamMsg {
			// 如果是嵌套map，序列化为JSON字符串
			if nestedMap, ok := value.(map[string]interface{}); ok {
				jsonData, err := json.Marshal(nestedMap)
				if err != nil {
					return err
				}
				msgMap[key] = string(jsonData)
			} else {
				msgMap[key] = value
			}
		}

		_, err := c.client.XAdd(c.ctx, &redis.XAddArgs{
			Stream: stream,
			Values: msgMap,
			MaxLen: maxLen,
			Approx: true, // 使用近似删除，提高性能
		}).Result()
		return err
	}

	// 对于其他类型，使用通用方法
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	var msgMap map[string]interface{}
	if err := json.Unmarshal(data, &msgMap); err != nil {
		return err
	}

	// 处理嵌套map
	for key, value := range msgMap {
		if nestedMap, ok := value.(map[string]interface{}); ok {
			jsonData, err := json.Marshal(nestedMap)
			if err != nil {
				return err
			}
			msgMap[key] = string(jsonData)
		}
	}

	_, err = c.client.XAdd(c.ctx, &redis.XAddArgs{
		Stream: stream,
		Values: msgMap,
		MaxLen: maxLen,
		Approx: true, // 使用近似删除，提高性能
	}).Result()

	return err
}

func (c *Client) CreateConsumerGroup(stream, group string) error {
	_, err := c.client.XGroupCreateMkStream(c.ctx, stream, group, "$").Result()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

func (c *Client) ReadMessages(stream, group, consumer string, count, block int64) ([]redis.XMessage, error) {
	xstreams, err := c.client.XReadGroup(c.ctx, &redis.XReadGroupArgs{
		Group:    group,
		Consumer: consumer,
		Streams:  []string{stream, ">"},
		Count:    count,
		Block:    time.Duration(block),
	}).Result()

	if err != nil {
		return nil, err
	}

	var messages []redis.XMessage
	for _, xstream := range xstreams {
		messages = append(messages, xstream.Messages...)
	}

	return messages, nil
}

func (c *Client) AcknowledgeMessage(stream, group, id string) error {
	return c.client.XAck(c.ctx, stream, group, id).Err()
}

func (c *Client) SetKey(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err=c.client.Set(c.ctx, key, string(data), expiration).Err()
	return err
}

func (c *Client) GetKey(key string, dest any) error {
	data, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (c *Client) DeleteKey(key string) error {
	return c.client.Del(c.ctx, key).Err()
}
