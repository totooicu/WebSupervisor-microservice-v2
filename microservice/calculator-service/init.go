package main

import (
	"log"

	"github.com/totooicu/go-mytool/stream"
)

func initService() {
	if err := stream.LoadConfig(); err != nil {
		log.Fatal("load config error:", err)
	}
	if err := stream.Init(); err != nil {
		log.Fatal("stream init error:", err)
	}
}