package actions

import (
	"log"
	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/go-mytool/json"
)



type CrawlerParameter struct {
	URL        string                 `json:"url"`
	Method     string                 `json:"method"`
	Headers    map[string]string      `json:"headers"`
	Body       map[string]interface{} `json:"body"`
	StrPayload string                 `json:"str_payload"`
}
// stream.RegisterService("http_request", services.HandleHttpRequestHttpRequest)
var cfg1=CrawlerParameter{
	URL:        "https://gradschool.zstu.edu.cn/",
	Method:     "GET",
	Headers:    map[string]string{},
	Body:       map[string]interface{}{},
	StrPayload: "",
}

func HttpRequestHttpRequest(cfg *CrawlerParameter){
	streamMsg,err := stream.Send("crawler-stream", "http_request", json.StructToMap(cfg), 1000*60*2)
	if err!= nil{
		log.Printf("Error - Send message to stream: %v", err)
		return
	}
	log.Printf("Debug - Send message to stream: %+v", streamMsg.Playload)


}