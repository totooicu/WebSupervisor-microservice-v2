package stream

import (
	"encoding/json"
	"fmt"
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
    ConsumerStream               string `json:"consumer_stream"`
    ConsumerGroup      string `json:"consumer_group"`
    GoroutineNum       int    `json:"goroutine_num"`
    GetTimeoutMs       int    `json:"get_timeout_ms"`   // 单位：毫秒
    MaxMessageBytes    int    `json:"max_message_bytes"` // 单位：字节
	CacheKeyPrefix     string `json:"cache_key_prefix"`
}


// Global config variables
var (
	cfg          Config
	StreamName   string
	ServiceName  string
	ConsumerGroup string
	MaxMsgSize   int
	Custom       map[string]interface{}
	CacheKeyPrefix string
	)


// LoadConfig 从 JSON 文件读取配置，并解析 ${} 环境变量
func LoadConfig(path string) error {
	//从命令行参数"-config"中获取路径
	args:=os.Args
	if len(args) == 1 {
		fmt.Printf("使用默认参数：%s\n",path)
	}else{
		path = args[1]
		fmt.Printf("使用自定义参数：%s\n",args[1])
	}
	


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
	StreamName = cfg.Stream.ConsumerStream
	ServiceName = cfg.Stream.ConsumerStream // 简化
	ConsumerGroup = cfg.Stream.ConsumerGroup
	MaxMsgSize = cfg.Stream.MaxMessageBytes
	if MaxMsgSize <= 0 {
		MaxMsgSize = 1024 * 1024 // 默认1MB
	}
	Custom = cfg.Custom
	CacheKeyPrefix = cfg.Stream.CacheKeyPrefix
	if CacheKeyPrefix == "" {
		CacheKeyPrefix = "stream:"
	}
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