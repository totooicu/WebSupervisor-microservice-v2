package stream

import (
	"encoding/json"
	"sync"
	"time"
	"fmt"
	"github.com/go-redis/redis/v8"
)

type ShardingAssembler struct {
	mu       sync.Mutex
	total    int
	received map[int][]byte
	done     chan *StreamMessage
}

func NewShardingAssembler(total int) *ShardingAssembler {
	return &ShardingAssembler{
		total:    total,
		received: make(map[int][]byte, total),
		done:     make(chan *StreamMessage, 1),
	}
}

func (sa *ShardingAssembler) Add(msg *StreamMessage) {
	sa.mu.Lock()
	defer sa.mu.Unlock()
	sa.received[msg.Sharding.ID] = msg.ShardingData
	if len(sa.received) == sa.total {
		data := make([]byte, 0)
		for i := 0; i < sa.total; i++ {
			data = append(data, sa.received[i]...)
		}
		fullMsg := &StreamMessage{}
		if err := json.Unmarshal(data, fullMsg); err != nil {
			// fallback
			fullMsg = &StreamMessage{
				MessageID:      msg.MessageID,
				ReplyID:        msg.ReplyID,
				ServiceName:    msg.ServiceName,
				CallbackStream: msg.CallbackStream,
				SourceStream:   msg.SourceStream,
				SourceService:  msg.SourceService,
				Sharding:       ShardingInfo{Total: 0},
				ErrCode:        msg.ErrCode,
				ErrMsg:         msg.ErrMsg,
				Deadline:       msg.Deadline,
				TraceID:        msg.TraceID,
				Playload:       map[string]interface{}{"_raw": data},
			}
		}
		sa.done <- fullMsg
		close(sa.done)
	}
}

func (sa *ShardingAssembler) Done() <-chan *StreamMessage {
	return sa.done
}

type ShardManager struct {
	mu         sync.Mutex
	assemblers map[string]*ShardingAssembler
	outChan    chan<- *StreamMessage
}

func NewShardManager(outChan chan<- *StreamMessage) *ShardManager {
	return &ShardManager{
		assemblers: make(map[string]*ShardingAssembler),
		outChan:    outChan,
	}
}

func (sm *ShardManager) Add(msg *StreamMessage) {
	key := msg.MessageID
	sm.mu.Lock()
	assembler, ok := sm.assemblers[key]
	if !ok {
		assembler = NewShardingAssembler(msg.Sharding.Total)
		sm.assemblers[key] = assembler
		go func(k string, as *ShardingAssembler) {
			select {
			case fullMsg := <-as.Done():
				sm.outChan <- fullMsg
				sm.mu.Lock()
				delete(sm.assemblers, k)
				sm.mu.Unlock()
			case <-time.After(30 * time.Second):
				sm.mu.Lock()
				delete(sm.assemblers, k)
				sm.mu.Unlock()
			}
		}(key, assembler)
	}
	sm.mu.Unlock()
	assembler.Add(msg)
}

// sendSharded 将大数据消息分片发送到目标 Stream
// data 为整个消息序列化后的字节切片
func sendSharded(msg *StreamMessage, data []byte,  targetStream string) error {
    chunkSize := cfg.Stream.MaxMessageBytes
    total := (len(data) + chunkSize - 1) / chunkSize

    for i := 0; i < total; i++ {
        start := i * chunkSize
        end := start + chunkSize
        if end > len(data) {
            end = len(data)
        }

        shardMsg := &StreamMessage{
            MessageID:      msg.MessageID,
            ReplyID:        msg.ReplyID,
            ServiceName:    msg.ServiceName,
            CallbackStream: msg.CallbackStream,
            SourceStream:   msg.SourceStream,
            SourceService:  msg.SourceService,
            Sharding: ShardingInfo{
                ID:    i,
                Total: total,
            },
            ShardingData: data[start:end],
            ErrCode:      msg.ErrCode,
            ErrMsg:       msg.ErrMsg,
            Deadline:     msg.Deadline,
            TraceID:      msg.TraceID,
        }

        if err := redisClient.XAdd(ctx, &redis.XAddArgs{
            Stream: targetStream, // 使用目标 Stream，而不是 CallbackStream
            Values: shardMsg.ToMap(),
        }).Err(); err != nil {
            return fmt.Errorf("send shard %d/%d failed: %w", i+1, total, err)
        }
    }
    return nil
}