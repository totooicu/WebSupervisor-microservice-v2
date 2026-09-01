package models
import (
	"github.com/totooicu/go-mytool/redis"
)

var REDIS_CONFIG RedisConfig
var REDIS_CLIENT *redis.Client
var CACHE_SERVICE *CacheService

type CacheService struct {
	redisClient *redis.Client
	debug       bool
}

 type RedisConfig struct {
    Addr string `json:"addr"`
    Password string `json:"password"`
    DB int `json:"db"`
   }

// CacheParameter 缓存服务参数结构体
type CacheParameter struct {
	App  string      `json:"app"`
	Key  string      `json:"key"`
	Data any `json:"data"`
}