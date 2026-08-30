package main

import (
	"log"
	"github.com/totooicu/go-mytool/file"
	"github.com/totooicu/go-mytool/json"
	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/web_supervisor-manager/models"
)

func AnyToX[T any](data any) T {
    // 使用带 ok 的类型断言，避免 panic
    if v, ok := data.(T); ok {
        return v
    }
    // 返回 T 的零值（适用于任何类型）
    var zero T
    return zero
}
func AnyArrToXs[T any](data []any) []T {
	var xs []T
	for _, v := range data {
		xs = append(xs, AnyToX[T](v))
	}
	return xs
}
func initapp(){
	//初始化stream
	models.CRAWLER_SERVICE = stream.Custom["crawler-service_consumer_stream"].(string)
	models.EMAIL_SERVICE = stream.Custom["email-service_consumer_stream"].(string)
	models.PARSER_SERVICE = stream.Custom["parser-service_consumer_stream"].(string)
	models.REDIS_CACHE_SERVICE = stream.Custom["redis_cache-service_consumer_stream"].(string)
	models.REDIS_APP_NAME = stream.Custom["redis_app_name"].(string)
	models.EMAIL_TOS = AnyArrToXs[string](stream.Custom["email_tos"].([]any))
{	interval_second:=stream.Custom["interval_second"].(float64)
	models.INTERVAL_SECONDS = int(interval_second)}
	
	log.Printf(">>>EMAIL_TOS: %v", models.EMAIL_TOS)
		
	//初始化jobs
	jobs_path:=stream.Custom["jobs_path"].(string)
	jobs_string := file.Read(jobs_path)
	log.Printf(">>>jobs_string: %v", len(jobs_string))
	// log.Printf(">>>jobs_navigator: %v", json.NewJSONNavigator(jobs_string).Get([]any{"urls"}).Result())
	jobs_arr_any := json.NewJSONNavigator(jobs_string).Get([]any{"urls"}).Result().([]any)
	log.Printf(">>>jobs_arr_any: %v\n", len(jobs_arr_any))
	models.JOBS = make([]models.Job, len(jobs_arr_any))
	for i := 0; i < len(jobs_arr_any); i++ {
		job:=json.AnyToMap(jobs_arr_any[i])
		header_string_any:=AnyToX[map[string]any](job["header"])
		header_string_string:=map[string]string{}
		for k,v:=range header_string_any{
			header_string_string[k]=v.(string)
		}
		body_string_any:=AnyToX[map[string]any](job["body"])
		body_string_string:=map[string]string{}
		for k,v:=range body_string_any{
			body_string_string[k]=v.(string)
		}
		crawler:=&models.CrawlerParameter{
			URL:         AnyToX[string](job["url"]),
			Method:     AnyToX[string](job["method"]),
			Header:    header_string_string,
			Body:       AnyToX[map[string]any](job["body"]),
			StrPayload: AnyToX[string](job["str_payload"]),
		}
		// log.Printf(">>>job: %#v", job["htmlKeys"].([]any)[0].(map[string]any)["key"])

		if job["htmlKeys"]==nil{job["htmlKeys"]=[]any{}}
		if job["jsonKeys"]==nil{job["jsonKeys"]=[]any{}}
		if job["xPathKeys"]==nil{job["xPathKeys"]=[]any{}}

		var hkeys, jkeys, xkeys []map[string]any
		hkeys=make([]map[string]any,len(job["htmlKeys"].([]any)))
		jkeys=make([]map[string]any,len(job["jsonKeys"].([]any)))
		xkeys=make([]map[string]any,len(job["xPathKeys"].([]any)))

		for k,v:=range job["htmlKeys"].([]any){hkeys[k]=v.(map[string]any)}
		for k,v:=range job["jsonKeys"].([]any){jkeys[k]=v.(map[string]any)}
		for k,v:=range job["xPathKeys"].([]any){xkeys[k]=v.(map[string]any)}
			
		
		keys:=&models.ParserParameter{Content:"", 
		HTMLKeys: hkeys, 
		JSONKeys: jkeys, 
		XPathKeys: xkeys}
		models.JOBS[i] = models.Job{Crawler: *crawler, Keys: *keys,Caches: []models.CacheParameter{}}
	}
	
	// log.Printf(">>>JOBS: %v", (json.AnyToMap(jobs_arr_any[0])))
	log.Printf(">>>JOBS: %v", models.JOBS[0])
	
}

func initClient() {
	if err := stream.LoadConfig("config.json"); err != nil {
		log.Fatal("load config error:", err)
	}
	if err := stream.Init(); err != nil {
		log.Fatal("stream init error:", err)
	}
	initapp()
}