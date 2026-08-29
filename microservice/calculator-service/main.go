package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/calculator-service/services"

)

func main() {
	initService()

	// 注册服务（不在 action 中）
	stream.RegisterService("add", services.HandleAdd)
	// stream.RegisterService("subtract", services.HandleSubtract)
	// stream.RegisterService("multiply", services.HandleMultiply)
	stream.RegisterService("divide", services.HandleDivide)

	log.Println("Calculator service started")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down...")
}