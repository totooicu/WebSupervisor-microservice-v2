package main

import (
	"log"

	"github.com/totooicu/email-service/models"
	"github.com/totooicu/go-mytool/stream"
	myjson "github.com/totooicu/go-mytool/json"
)
func initEmailConfig() {
	 e:= myjson.AnyToStruct(stream.Custom["emails"], &models.EMAILS)
	 if e!= nil {
		 log.Fatal("email config error:", e)
	}
	
	models.DEFAULT_EMAIL = stream.Custom["default_email"].(string)
}
func initService() {
	if err := stream.LoadConfig(); err != nil {
		log.Fatal("load config error:", err)
	}
	if err := stream.Init(); err != nil {
		log.Fatal("stream init error:", err)
	}
	initEmailConfig()
}