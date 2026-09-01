package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/redis_cache-service/services"

)

func main() {
	initService()

	// 注册服务（不在 action 中）
	stream.RegisterService("compare_and_save", services.HandleCompareAndSave)
	stream.RegisterService("get", services.HandleGet)
	stream.RegisterService("set", services.HandleSet)
	stream.RegisterService("delete", services.HandleDelete)
	stream.RegisterService("get_and_set", services.HandleGetAndSet)

	log.Println("Calculator service started")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}