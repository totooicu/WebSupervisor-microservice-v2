# Stream开发工具包

一个基于Redis Streams的微服务通信开发工具包，提供统一的消息传输、请求响应和服务管理机制。

## 功能特性

- ✅ 基于Redis Streams的可靠消息传输
- ✅ 请求-响应模式，支持阻塞获取响应
- ✅ 消息网关，统一路由和分发
- ✅ 服务注册和管理
- ✅ 支持超时机制和错误处理
- ✅ 线程安全设计
- ✅ 自动清理过期响应通道

## 目录结构

```
streamtool/
├── models/          # 数据模型定义
│   └── models.go
├── response/        # 响应管理
│   └── response.go
├── example/         # 使用示例
│   └── example.go
├── streamtool.go    # 核心工具类
├── errors.go        # 错误定义
└── README.md        # 说明文档
```

## 核心组件

### 1. StreamTool

核心工具类，提供以下功能：

- `InitStreamTool(config)` - 初始化工具
- `GetStreamTool()` - 获取工具实例
- `StreamPush(msg, stream)` - 推送消息到stream
- `StreamGet(stream, group, consumer)` - 从stream获取消息
- `Send(msg, stream)` - 发送消息并返回响应对象
- `StartGateway(stream, group, consumer)` - 启动消息网关
- `StartService(serviceName, handler)` - 启动单个服务
- `StartAllServices(serviceHandlers)` - 启动所有服务

### 2. Response

响应对象，提供：

- `Get()` - 阻塞获取响应
- `GetWithTimeout(timeout)` - 带超时获取响应

### 3. 消息网关

自动处理消息路由：
- 服务不存在时返回错误
- 响应消息自动分发到对应通道
- 业务消息分发到对应服务

## 使用示例

### 基本用法

```go
package main

import (
	"log"
	
	"WebSupervisor/MyTool/streamtool"
	"WebSupervisor/MyTool/streamtool/models"
)

func main() {
	// 1. 配置初始化
	config := &models.StreamToolConfig{
		Redis: models.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		Services: []models.ServiceConfig{
			{
				Name:          "my_service",
				StreamName:    "my-stream",
				ConsumerGroup: "my-group",
				ConsumerID:    "my-consumer-1",
			},
		},
		Debug: true,
	}

	// 2. 初始化Stream工具
	streamtool.InitStreamTool(config)
	st := streamtool.GetStreamTool()

	// 3. 启动消息网关
	st.StartGateway("input-stream", "gateway-group", "gateway-consumer")

	// 4. 定义服务处理函数
	serviceHandlers := map[string]func(msg *models.StreamMessage){
		"my_service": myServiceHandler,
	}

	// 5. 启动所有服务
	st.StartAllServices(serviceHandlers)

	log.Println("All services started successfully")

	// 6. 发送消息并等待响应
	requestMsg := &models.StreamMessage{
		ServiceName:    "my_service",
		CallbackStream: "response-stream",
		Playload: map[string]interface{}{
			"key": "value",
		},
	}

	response := st.Send(requestMsg, "input-stream")
	if response != nil {
		result := response.Get()
		log.Printf("Received response: %v", result)
	}

	// 保持程序运行
	select {}
}

func myServiceHandler(msg *models.StreamMessage) {
	log.Printf("Processing service: %+v", msg)
	
	// 处理业务逻辑
	result := map[string]interface{}{
		"status": "success",
		"data":   "processed",
	}
	
	st := streamtool.GetStreamTool()
	st.SendResponse(msg.MessageID, msg.CallbackStream, result)
}
```

### 服务嵌套调用

```go
func getmidServiceHandler(msg *models.StreamMessage) {
	st := streamtool.GetStreamTool()
	
	// 调用自己的其他服务
	subMsg := &models.StreamMessage{
		ServiceName:    "find_service",
		CallbackStream: "", // 不发送stream
		Playload: map[string]interface{}{
			"text": "Hello World",
			"pattern": "o",
		},
	}
	
	// 阻塞获取结果
	subResult := st.Send(subMsg, "input-stream").Get()
	
	result := map[string]interface{}{
		"sub_result": subResult,
	}
	
	st.SendResponse(msg.MessageID, msg.CallbackStream, result)
}
```

## 消息结构

```go
type StreamMessage struct {
	MessageID       string                 `json:"message_id"`
	ReplyID         string                 `json:"reply_id"`
	ServiceName     string                 `json:"service_name"`
	CallbackStream  string                 `json:"callback_stream"`
	Playload        map[string]interface{} `json:"playload"`
}
```

## 错误处理

工具包提供以下错误：
- `ErrResponseChannelNotFound` - 响应通道未找到
- `ErrResponseTimeout` - 响应超时
- `ErrServiceNotFound` - 服务未找到
- `ErrStreamPushFailed` - 消息推送失败
- `ErrStreamGetFailed` - 消息获取失败

## 配置说明

- **Redis配置**：指定Redis服务器地址和认证信息
- **服务配置**：定义服务名称、stream名称、消费者组和消费者ID
- **Debug模式**：开启调试日志

## 最佳实践

1. **服务设计**：每个服务专注于单一功能
2. **错误处理**：使用带超时的响应获取
3. **资源管理**：定期清理过期通道
4. **监控**：实现健康检查和性能监控

## 性能优化

- 使用批量消息处理
- 合理设置超时时间
- 监控响应通道数量
- 实现服务熔断机制

## 依赖

- github.com/go-redis/redis/v8
- WebSupervisor/MyTool/mysync

## License

MIT