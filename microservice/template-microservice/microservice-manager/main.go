package main

import (
	"log"
	"time"
	"github.com/totooicu/microservice-manager/actions"

)

func main() {
	initClient()
	log.Println("Client started")
	// 执行调用
	actions.Action()
	// 防止进程立即退出
	time.Sleep(2 * time.Second)
}