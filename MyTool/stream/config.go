package stream

import (
	"encoding/json"

	"os"
	"regexp"

)

// Config 通信库配置
type Config struct {
	Redis        RedisConfig        `json:"redis"`
	Stream       StreamConfig       `json:"stream"`
	Custom       map[string]interface{} `json:"custom"`
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type StreamConfig struct {
    Name               string `json:"name"`
    ConsumerGroup      string `json:"consumer_group"`
    GoroutineNum       int    `json:"goroutine_num"`
    GetTimeoutMs       int    `json:"get_timeout_ms"`   // 单位：毫秒
    MaxMessageBytes    int    `json:"max_message_bytes"` // 单位：字节
}

// Global config variables
var (
	cfg          Config
	StreamName   string
	ServiceName  string
	ConsumerGroup string
	MaxMsgSize   int
	Custom       map[string]interface{}
)

// LoadConfig 从 JSON 文件读取配置，并解析 ${} 环境变量
func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// 先替换环境变量
	expanded := expandEnvVars(string(data))
	if err := json.Unmarshal([]byte(expanded), &cfg); err != nil {
		return err
	}

	// 设置全局变量
	StreamName = cfg.Stream.Name
	ServiceName = cfg.Stream.Name // 简化
	ConsumerGroup = cfg.Stream.ConsumerGroup
	MaxMsgSize = cfg.Stream.MaxMessageBytes
	if MaxMsgSize <= 0 {
		MaxMsgSize = 1024 * 1024 // 默认1MB
	}
	Custom = cfg.Custom
	return nil
}

// expandEnvVars 替换字符串中的 ${VAR} 为环境变量值
func expandEnvVars(s string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		// 提取变量名
		varName := match[2 : len(match)-1]
		return os.Getenv(varName)
	})
}