package actions
import (
	"log"

	"github.com/totooicu/go-mytool/stream"
	 "github.com/totooicu/web_supervisor-manager/models"
)
// type CacheParameter struct {
// 	App  string      `json:"app"`
// 	Key  string      `json:"key"`
// 	Data interface{} `json:"data"`
// }

	// stream.RegisterService("compare_and_save", services.HandleCompareAndSave)
	// stream.RegisterService("get", services.HandleGet)
	// stream.RegisterService("set", services.HandleSet)
	// stream.RegisterService("delete", services.HandleDelete)
	// stream.RegisterService("get_and_set", services.HandleGetAndSet)


func compareAndSave(app string,key string,data any)   {
	resp, err := stream.Send(models.REDIS_CACHE_SERVICE, "compare_and_save", map[string]interface{}{
		"app": app,
		"key": key,
		"data": data,
	}, 0)
	if err != nil {
		log.Fatal("send compare_and_save error:", err)
	}
	if resp.ErrMsg != "" {
		log.Fatal("compare_and_save error:", resp.ErrMsg)
	}
	log.Println("compare_and_save resp:", resp.Playload)
}

func get(app string,key string)   {
	resp, err := stream.Send(models.REDIS_CACHE_SERVICE, "get", map[string]interface{}{
		"app": app,
		"key": key,
	}, 0)
	if err != nil {
		log.Fatal("send get error:", err)
	}
	log.Println("get resp:", resp.Playload)
}

func set(app string,key string,data any)   {
	resp, err := stream.Send(models.REDIS_CACHE_SERVICE, "set", map[string]interface{}{
		"app": app,
		"key": key,
		"data": data,
	}, 0)
	if err != nil {
		log.Fatal("send set error:", err)
	}
	log.Println("set resp:", resp.Playload)
}
func delete(app string,key string)   {
	resp, err := stream.Send(models.REDIS_CACHE_SERVICE, "delete", map[string]interface{}{
		"app": app,
		"key": key,
	}, 0)
	if err != nil {
		log.Fatal("send delete error:", err)
	}
	log.Println("delete resp:", resp.Playload)
}
func getAndSet(app string,key string,data any)   {
	resp, err := stream.Send(models.REDIS_CACHE_SERVICE, "get_and_set", map[string]interface{}{
		"app": app,
		"key": key,
		"data": data,
	}, 0)
	if err != nil {
		log.Fatal("send get_and_set error:", err)
	}
	log.Println("get_and_set resp:", resp.Playload)
}