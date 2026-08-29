package stream

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"strings"

	"github.com/go-redis/redis/v8"
)

var (
	HANDELS           = make(map[string]HandlerFunc)
	RESPONSES         = make(map[string]chan *StreamMessage)
	SHARDING_RESPONSES = make(map[string]*ShardingAssembler)
	mutex             sync.RWMutex
	semaphore         chan struct{}
	consumerName      string
	activeHandlers    int32 // 当前正在执行的业务 handler 数量
)

type HandlerFunc func(msg *StreamMessage)

func RegisterService(name string, handler HandlerFunc) {
	mutex.Lock()
	defer mutex.Unlock()
	HANDELS[name] = handler
}

// Init 初始化库并启动消费者
func Init() error {
	if err := InitClient(); err != nil {
		return err
	}
	consumerName = fmt.Sprintf("%s-%d", ServiceName, time.Now().UnixNano())
	semaphore = make(chan struct{}, cfg.Stream.GoroutineNum)

	// 注册内置响应处理
	RegisterService("response", handleResponse)

	// 确保消费者组存在（如果 Stream 不存在则自动创建）
	if err := ensureConsumerGroup(); err != nil {
		return err
	}

	go consumeLoop()
	return nil
}

// ensureConsumerGroup 创建消费者组，如果组已存在则忽略错误
func ensureConsumerGroup() error {
	err := redisClient.XGroupCreateMkStream(ctx, StreamName, ConsumerGroup, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func consumeLoop() {
	reassembledChan := make(chan *StreamMessage, 100)
	shardManager := NewShardManager(reassembledChan)

	for {
		// 优先处理重组完成的消息
		select {
		case msg := <-reassembledChan:
			semaphore <- struct{}{}
			go func(m *StreamMessage) {
				defer func() { <-semaphore }()
				dispatch(m)
			}(msg)
			continue
		default:
		}

		streams, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    ConsumerGroup,
			Consumer: consumerName,
			Streams:  []string{StreamName, ">"},
			Count:    1,
			Block:    0,
		}).Result()
		if err != nil {
			log.Println("XReadGroup error:", err)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, message := range stream.Messages {
				msg, err := FromMap(message.Values)
				if err != nil {
					log.Println("parse error:", err)
					redisClient.XAck(ctx, StreamName, ConsumerGroup, message.ID)
					continue
				}

				// 特殊处理 ping 消息：不占用信号量，直接启动新协程
				if msg.ServiceName == "ping" {
					go func(m *StreamMessage, msgID string) {
						handlePing(m)
						redisClient.XAck(ctx, StreamName, ConsumerGroup, msgID)
					}(msg, message.ID)
					continue
				}

				// 分片消息
				if msg.Sharding.Total > 0 {
					shardManager.Add(msg)
					redisClient.XAck(ctx, StreamName, ConsumerGroup, message.ID)
					continue
				}

				// 超时检查（ping 消息不设超时）
				if msg.Deadline > 0 && msg.ServiceName != "response" {
					if time.UnixMilli(msg.Deadline).Before(time.Now()) {
						redisClient.XAck(ctx, StreamName, ConsumerGroup, message.ID)
						continue
					}
				}

				// 普通业务消息：获取信号量，然后启动 goroutine
				semaphore <- struct{}{}
				go func(m *StreamMessage, msgID string) {
					defer func() { <-semaphore }()
					dispatch(m)
					redisClient.XAck(ctx, StreamName, ConsumerGroup, msgID)
				}(msg, message.ID)
			}
		}
	}
}

// dispatch 调用对应的 handler，并统计活跃协程数
func dispatch(msg *StreamMessage) {
	atomic.AddInt32(&activeHandlers, 1)
	defer atomic.AddInt32(&activeHandlers, -1)

	mutex.RLock()
	handler, ok := HANDELS[msg.ServiceName]
	mutex.RUnlock()
	if !ok {
		log.Printf("unknown service: %s\n", msg.ServiceName)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic in %s: %v\n", msg.ServiceName, r)
		}
	}()
	handler(msg)
}

// 获取当前协程使用情况（供 ping 响应使用）
func getGoroutineStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["active_handlers"] = atomic.LoadInt32(&activeHandlers)
	stats["total_goroutines"] = runtime.NumGoroutine()
	stats["capacity"] = cap(semaphore)
	stats["available"] = cap(semaphore) - len(semaphore) // 使用 len 更准确
	return stats
}