package main

import (
	"log"
	"github.com/totooicu/go-mytool/redis"

	"github.com/totooicu/redis_cache-service/models"
	"github.com/totooicu/go-mytool/stream"
	myjson "github.com/totooicu/go-mytool/json"
)
func initCacheConfig() {
	
	 e:= myjson.AnyToStruct(stream.Custom["redis"], &models.REDIS_CONFIG)
	 if e!= nil {
		 log.Fatal("cache config error:", e)
	}
	if models.REDIS_CONFIG.Addr == "" {
		models.REDIS_CONFIG=stream.Custom["redis"].(models.RedisConfig)
	}
	models.REDIS_CLIENT=redis.NewClient(models.REDIS_CONFIG.Addr,-1,models.REDIS_CONFIG.Password,	models.REDIS_CONFIG.DB,
	)

	
}
func initService() {
	if err := stream.LoadConfig("config.json"); err != nil {
		log.Fatal("load config error:", err)
	}
	if err := stream.Init(); err != nil {
		log.Fatal("stream init error:", err)
	}
	initCacheConfig()
}