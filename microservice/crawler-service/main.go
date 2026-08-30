package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/crawler-service/services"

)

func main() {
	initService()

	// 注册服务（不在 action 中）
	stream.RegisterService("http_request", services.HandleHttpRequestHttpRequest)


	log.Println("Calculator service started")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}