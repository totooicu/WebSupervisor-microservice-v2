package stream

import (

	"time"
)

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