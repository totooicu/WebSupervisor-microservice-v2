package stream

import (
	"encoding/json"
	"strconv"
)

// ShardingInfo 分片信息
type ShardingInfo struct {
	ID    int `json:"id"`
	Total int `json:"total"`
}

// StreamMessage 流消息结构
type StreamMessage struct {
	MessageID      string                 `json:"message_id"`
	ReplyID        string                 `json:"reply_id"`
	ServiceName    string                 `json:"service_name"`
	CallbackStream string                 `json:"callback_stream"`
	SourceStream   string                 `json:"source_stream"`
	SourceService  string                 `json:"source_service"`
	Playload       map[string]interface{} `json:"playload"`
	Sharding       ShardingInfo           `json:"sharding"`
	ShardingData   []byte                 `json:"sharding_data,omitempty"`
	ErrCode        int                    `json:"err_code,omitempty"`
	ErrMsg         string                 `json:"err_msg,omitempty"`
	Deadline       int64                  `json:"deadline,omitempty"`  // 绝对截止时间戳（毫秒），0表示不限
	TraceID        string                 `json:"trace_id,omitempty"`
}

// ToMap 将消息转换为 Redis Stream 可接受的扁平 map
func (m *StreamMessage) ToMap() map[string]interface{} {
	// 将 Playload 序列化为 JSON 字符串
	payloadBytes, _ := json.Marshal(m.Playload)
	result := map[string]interface{}{
		"message_id":       m.MessageID,
		"reply_id":         m.ReplyID,
		"service_name":     m.ServiceName,
		"callback_stream":  m.CallbackStream,
		"source_stream":    m.SourceStream,
		"source_service":   m.SourceService,
		"playload":         string(payloadBytes),
		"sharding_id":      m.Sharding.ID,
		"sharding_total":   m.Sharding.Total,
		"err_code":         m.ErrCode,
		"err_msg":          m.ErrMsg,
		"deadline":         m.Deadline,
		"trace_id":         m.TraceID,
	}
	if m.ShardingData != nil {
		result["sharding_data"] = m.ShardingData // []byte 类型，go-redis 可处理
	}
	return result
}

// FromMap 从 Redis Stream 读取的 map 解析出 StreamMessage
func FromMap(values map[string]interface{}) (*StreamMessage, error) {
	msg := &StreamMessage{}

	// 辅助函数：获取字符串字段
	getString := func(key string) string {
		if v, ok := values[key]; ok {
			switch val := v.(type) {
			case string:
				return val
			case []byte:
				return string(val)
			}
		}
		return ""
	}
	// 辅助函数：获取整数字段
	getInt := func(key string) int {
		if v, ok := values[key]; ok {
			switch val := v.(type) {
			case int:
				return val
			case int64:
				return int(val)
			case float64:
				return int(val)
			case string:
				n, _ := strconv.Atoi(val)
				return n
			}
		}
		return 0
	}
	// 辅助函数：获取 int64 字段
	getInt64 := func(key string) int64 {
		if v, ok := values[key]; ok {
			switch val := v.(type) {
			case int64:
				return val
			case int:
				return int64(val)
			case float64:
				return int64(val)
			case string:
				n, _ := strconv.ParseInt(val, 10, 64)
				return n
			}
		}
		return 0
	}

	msg.MessageID = getString("message_id")
	msg.ReplyID = getString("reply_id")
	msg.ServiceName = getString("service_name")
	msg.CallbackStream = getString("callback_stream")
	msg.SourceStream = getString("source_stream")
	msg.SourceService = getString("source_service")
	msg.ErrMsg = getString("err_msg")
	msg.TraceID = getString("trace_id")

	// 解析 playload（JSON 字符串 -> map）
	payloadStr := getString("playload")
	if payloadStr != "" {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(payloadStr), &payload); err == nil {
			msg.Playload = payload
		} else {
			// 解析失败则保留空 map，避免 nil
			msg.Playload = make(map[string]interface{})
		}
	} else {
		msg.Playload = make(map[string]interface{})
	}

	// 分片信息
	msg.Sharding.ID = getInt("sharding_id")
	msg.Sharding.Total = getInt("sharding_total")

	// 分片数据（可能是 []byte 或 string）
	if v, ok := values["sharding_data"]; ok {
		switch val := v.(type) {
		case []byte:
			msg.ShardingData = val
		case string:
			msg.ShardingData = []byte(val)
		}
	}

	msg.ErrCode = getInt("err_code")
	msg.Deadline = getInt64("deadline")

	return msg, nil
}