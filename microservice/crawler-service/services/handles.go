package services

import (
	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/crawler-service/models"
	log "github.com/sirupsen/logrus"
	"encoding/json"
	"github.com/totooicu/go-mytool/http"
)

func HandleHttpRequestHttpRequest(msg *stream.StreamMessage) {
	log.Printf(">>>rocessing msg.Playload: %v", len(msg.Playload))
	// 检查url字段是否存在
	if _, ok := msg.Playload["url"]; !ok {
		log.Printf("Error: url field not found in playload")
		return
	}

	// 解析参数
	var params models.CrawlerParameter
	playloadData, err := json.Marshal(msg.Playload)
	if err != nil {
		log.Printf("Error marshalling playload: %v", err)
		return
	}

	if err := json.Unmarshal(playloadData, &params); err != nil {
		log.Printf("Error unmarshalling playload: %v", err)
		return
	}

	var response string
	var statusCode int

	httpClient := http.NewHttpHeader(params.URL, params.Headers)

	switch params.Method {
	case "GET":
		httpClient.Get("").Read()
		response = httpClient.GetBodyString()
		statusCode = httpClient.GetStatusCode()
	case "POST":
		httpClient.Post(params.Body, params.StrPayload).Read()
		response = httpClient.GetBodyString()
		statusCode = httpClient.GetStatusCode()
	default:
		log.Printf("Unsupported method: %s", params.Method)
		
		return
	}

	// 检查状态码是否有效
	if statusCode == 0 {
		log.Printf("Warning: Status code is 0, request may have failed")
	} else {
		log.Printf("Debug - HTTP response status code: %d", statusCode)
	}
	log.Printf(">>>handleHttpRequest response: %s", len(response))
	// 构造响应消息
	paramData := map[string]interface{}{
		"content": response,
		"status": statusCode,
	}
	e:=stream.ResponseSucc(msg, paramData)
	if e!= nil{
		log.Printf("Error response: %v", e)
		return
	}
}
