package models

// RedisConfig Redis配置
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// StreamMessage 流消息结构
type StreamMessage struct {
	MessageID      string                 `json:"message_id"`
	ReplyID        string                 `json:"reply_id"`
	ServiceName    string                 `json:"service_name"`
	CallbackStream string                 `json:"callback_stream"`
	SourceStream   string                 `json:"source_stream"`  // 源Stream名称
	SourceService  string                 `json:"source_service"` // 源服务名称
	Playload       map[string]interface{} `json:"playload"`
}

// ServiceConfig 服务配置
type ServiceConfig struct {
	Name          string `json:"name"`
	StreamName    string `json:"stream_name"`
	ConsumerGroup string `json:"consumer_group"`
	ConsumerID    string `json:"consumer_id"`
}

// StreamToolConfig Stream工具配置
type StreamToolConfig struct {
	Redis           RedisConfig     `json:"redis"`
	Services        []ServiceConfig `json:"services"`
	Debug           bool            `json:"debug"`
	MaxStreamLength int             `json:"max_stream_length"` // 最大消息数量限制
}
