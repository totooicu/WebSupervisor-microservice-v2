package main

import (
	"log"
	"time"
	"github.com/totooicu/web_supervisor-manager/actions"

)

func main() {
	initClient()
	log.Println("Client started")
	// 执行调用
	actions.Action()
	// 防止进程立即退出
	time.Sleep(2 * time.Second)
}