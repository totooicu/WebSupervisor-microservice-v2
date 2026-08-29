package main

import (
	"log"

	"github.com/totooicu/go-mytool/streamtool"
	"github.com/totooicu/go-mytool/streamtool/models"
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
				Name:          "find_service",
				StreamName:    "find-stream",
				ConsumerGroup: "find-group",
				ConsumerID:    "find-consumer-1",
			},
			{
				Name:          "getmid_service",
				StreamName:    "getmid-stream",
				ConsumerGroup: "getmid-group",
				ConsumerID:    "getmid-consumer-1",
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
		"find_service": findServiceHandler,
		"getmid_service": getmidServiceHandler,
	}

	// 5. 启动所有服务
	st.StartAllServices(serviceHandlers)

	log.Println("All services started successfully")

	// 6. 应用示例：调用服务
	// 创建请求消息
	requestMsg := &models.StreamMessage{
		ServiceName:    "find_service",
		CallbackStream: "response-stream",
		Playload: map[string]interface{}{
			"text": "Hello World",
			"pattern": "o",
		},
	}

	// 发送消息并等待响应
	response := st.Send(requestMsg, "input-stream")
	if response != nil {
		result := response.Get()
		log.Printf("Received response: %v", result)
	}

	// 保持程序运行
	select {}
}

// findServiceHandler find服务处理函数
func findServiceHandler(msg *models.StreamMessage) {
	log.Printf("Processing find_service: %+v", msg)
	
	// text := msg.Playload["text"].(string)
	// pattern := msg.Playload["pattern"].(string)
	
	// 模拟处理逻辑
	result := map[string]interface{}{
		"found": true,
		"positions": []int{4},
	}
	
	st := streamtool.GetStreamTool()
	st.SendResponse(msg.MessageID, msg.CallbackStream, result)
}

// getmidServiceHandler 取中间服务处理函数
func getmidServiceHandler(msg *models.StreamMessage) {
	log.Printf("Processing getmid_service: %+v", msg)
	
	// 调用自己的find_service
	st := streamtool.GetStreamTool()
	
	leftMsg := &models.StreamMessage{
		ServiceName:    "find_service",
		CallbackStream: "", // 不发送stream
		Playload: map[string]interface{}{
			"text": msg.Playload["text"].(string),
			"pattern": msg.Playload["left_pattern"].(string),
		},
	}
	
	rightMsg := &models.StreamMessage{
		ServiceName:    "find_service",
		CallbackStream: "",
		Playload: map[string]interface{}{
			"text": msg.Playload["text"].(string),
			"pattern": msg.Playload["right_pattern"].(string),
		},
	}
	
	// 使用直接返回获取结果
	leftResult := st.Send(leftMsg, "input-stream").Get()
	rightResult := st.Send(rightMsg, "input-stream").Get()
	
	result := map[string]interface{}{
		"left": leftResult,
		"right": rightResult,
	}
	
	st.SendResponse(msg.MessageID, msg.CallbackStream, result)
}