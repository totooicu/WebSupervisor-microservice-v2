package main

import (
	"log"
	"time"
)

func main() {
	initClient()
	log.Println("Client started")
	// 执行调用
	callCalculator()
	// 防止进程立即退出
	time.Sleep(2 * time.Second)
}