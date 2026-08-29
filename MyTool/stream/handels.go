package stream

import (
	"log"
	"time"
)


// handlePing 处理 ping 请求，返回当前协程使用情况
func handlePing(msg *StreamMessage) {
	stats := getGoroutineStats()
	err := Response(msg, stats, 0, "")
	if err != nil {
		log.Printf("ping response error: %v\n", err)
	}
}


func handleResponse(msg *StreamMessage) {
	if msg.Sharding.Total == 0 {
		mutex.RLock()
		ch, ok := RESPONSES[msg.ReplyID]
		mutex.RUnlock()
		if ok {
			ch <- msg
		}
		return
	}

	// 分片响应
	key := "sharding:" + msg.ReplyID
	mutex.Lock()
	assembler, ok := SHARDING_RESPONSES[key]
	if !ok {
		assembler = NewShardingAssembler(msg.Sharding.Total)
		SHARDING_RESPONSES[key] = assembler
		go func(k string, as *ShardingAssembler) {
			select {
			case fullMsg := <-as.Done():
				mutex.RLock()
				ch, exists := RESPONSES[fullMsg.ReplyID]
				mutex.RUnlock()
				if exists {
					ch <- fullMsg
				}
				mutex.Lock()
				delete(SHARDING_RESPONSES, k)
				mutex.Unlock()
			case <-time.After(30 * time.Second):
				mutex.Lock()
				delete(SHARDING_RESPONSES, k)
				mutex.Unlock()
			}
		}(key, assembler)
	}
	mutex.Unlock()
	assembler.Add(msg)
}


// func handelShard(msg *StreamMessage) {
// 	select {
// 		case rmsg := <-reassembledChan:
// 			semaphore <- struct{}{}
// 			go func(m *StreamMessage) {
// 				defer func() { <-semaphore }()
// 				dispatch(m)
// 			}(rmsg)
// 		default:
// 		}
	

// 	if msg.Sharding.Total > 0 {
// 		shardManager.Add(msg)
// 	}
// }

