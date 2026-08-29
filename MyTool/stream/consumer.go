package stream

import (

	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"strings"

	// "github.com/go-redis/redis/v8"
)

var (
	HANDELS           = make(map[string]HandlerFunc)
	RESPONSES         = make(map[string]chan *StreamMessage)
	SHARDING_RESPONSES = make(map[string]*ShardingAssembler)
	mutex             sync.RWMutex
	semaphore         chan struct{}
	consumerName      string
	activeHandlers    int32 // 当前正在执行的业务 handler 数量

	reassembledChan = make(chan *StreamMessage, 100)// 分片重组完成的消息通道
	shardManager = NewShardManager(reassembledChan)// 分片管理器

)

	
type HandlerFunc func(msg *StreamMessage)



// ensureConsumerGroup 创建消费者组，如果组已存在则忽略错误
func ensureConsumerGroup() error {
	err := redisClient.XGroupCreateMkStream(ctx, StreamName, ConsumerGroup, "$").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

func consumeLoop() {
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

		message,err:=ReadOne(ctx, consumerName)
		if err!= nil {
			log.Println("ReadOne error:", err)
			continue
		}
		msg, err := FromMap(message.Values)
		if err != nil {
			log.Println("parse error:", err)
			redisClient.XAck(ctx, StreamName, ConsumerGroup, message.ID)
			continue
		}
		//超时丢弃消息
		if msg.Deadline > 0&&time.UnixMilli(msg.Deadline).Before(time.Now()) {
			redisClient.XAck(ctx, StreamName, ConsumerGroup, message.ID)
			continue
		}
		
		// 特殊处理 ping 消息：不占用信号量，直接启动新协程
		if msg.ServiceName == "ping" {
			go func(m *StreamMessage, msgID string) {
				// handlePing(m)
				HANDELS["ping"](m)
				redisClient.XAck(ctx, StreamName, ConsumerGroup, msgID)
			}(msg, message.ID)
		
			continue
		}

		// 分片消息
		if msg.Sharding.Total > 0 {
			shardManager.Add(msg)
			// HANDELS["shard"](msg)
			redisClient.XAck(ctx, StreamName, ConsumerGroup, message.ID)
			continue
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