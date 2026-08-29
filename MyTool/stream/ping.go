package stream

import (
	"log"
)


// handlePing 处理 ping 请求，返回当前协程使用情况
func handlePing(msg *StreamMessage) {
	stats := getGoroutineStats()
	err := Response(msg, stats, 0, "")
	if err != nil {
		log.Printf("ping response error: %v\n", err)
	}
}
