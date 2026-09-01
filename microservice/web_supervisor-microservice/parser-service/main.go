package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/parser-service/services"

)

func main() {
	initService()

	// 注册服务（不在 action 中）
	stream.RegisterService("parse_html_by_xpath", services.HandleParseHTMLByXPath)
	stream.RegisterService("parse_html_by_get_mid", services.HandleParseHTMLByGetMid)
	stream.RegisterService("parse_json", services.HandleParseJSON)


	log.Println("Calculator service started")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}