package actions

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/totooicu/web_supervisor-manager/models"
	"github.com/totooicu/web_supervisor-manager/services"
)

func get_url_content(crawler *models.CrawlerParameter)(string,error){
		pld, err := services.Crawl(crawler)
	if err != nil || (pld["status"] != nil && pld["status"].(float64) != 200) {
		log.Warnf("Error - Crawl: %v | status: %v", err, pld["status"])
		services.ReportError(crawler.URL, "crawl", fmt.Sprintf("err=%v status=%v", err, pld["status"]))
		return "", err
		}
	return pld["content"].(string), nil
}
func get_conmamd_content(command *models.CommandParameter)(string,error){
		ctt,Err:=services.RunCommand(command)
	switch Err{
		case nil:
			break
		default://严重错误
			log.Warnf("Error - RunCommand: %v", Err)
			services.ReportError(command.Command, "execute", fmt.Sprintf("%v", Err))
			return "", Err
	}
	return ctt, nil
}

func run_one(job *models.Job) {
	// 任务级 panic 恢复：致命错误只跳过本次任务，不终止整个服务
	defer func() {
		if r := recover(); r != nil {
			log.Warnf("Error - run_one panic recovered (task skipped): %v", r)
			services.ReportError(job.Crawler.URL, "run_one", fmt.Sprintf("%v", r))
		}
	}()
	var err error
	//发送http请求
	switch job.JobType{
	case "url":
		job.Keys.Content, err = get_url_content(&job.Crawler) ;break
	case "command":
		 job.Keys.Content, err = get_conmamd_content(&job.Command) ;break
	}
	if err != nil {
		log.Warnf("Error - get_conmamd_content: %v", err)
		services.ReportError(job.Crawler.URL, "get_conmamd_content", err.Error())
		return
	}
	


	//解析html
	tgt, err := services.Parse(&job.Keys)
	if err != nil {
		log.Warnf("Error - Parse: %v", err)
		services.ReportError(job.Crawler.URL, "parse", err.Error())
		return
	}
	log.Printf(">>> Debug - tgt: %+v", len(tgt))
	pd := tgt["parsed_data"].(map[string]any) //->map[string][]any
	//缓存与匹配
	//准备数据
	newCaches := make([]models.CacheParameter, len(pd))
	i := 0
	changed_new_datas := []*models.CacheParameter{} //->map[key][old,new]
	for k, v := range pd {
		key := fmt.Sprintf("%s:%s", job.Crawler.URL, k)
		newCaches[i] = models.CacheParameter{models.REDIS_APP_NAME, key, v}
		log.Printf(">>> Debug - k: %s\n", newCaches[i].Key)
		r, e := services.CacheCompareAndSave(&newCaches[i])
		log.Printf(">>> Debug - r: %+v\n", r)
		if e != nil { //缓存不存在
			log.Warnf("Error - CacheCompareAndSave: %v\n", e)
			rr, e := services.CacheSet(&newCaches[i])
			if e != nil {
				log.Warnf("Error - CacheSet: %v\n", e)
				services.ReportError(job.Crawler.URL, "cache_set", e.Error())
				continue
			}

			if rr["error"] != nil && rr["error"].(string) != "" {
				log.Warnf("Error - CacheSet: %v\n", rr["error"])
				services.ReportError(job.Crawler.URL, "cache_set", rr["error"].(string))
				continue
			}
			changed_new_datas = append(changed_new_datas, &newCaches[i])
			continue
		}
		if r["changed"].(bool) { //记录变更的数据
			changed_new_datas = append(changed_new_datas, &newCaches[i])
		}
		i++
	}

	//if changed_new_datas is empty
	if len(changed_new_datas) == 0 {
		return
	}
	job.Caches = newCaches
	services.EmailNoticeChanged(&changed_new_datas, &job.Crawler.URL)

	log.Infof(">>>执行任务成功") //绿色

}
func runs() {
	var i int64 = 0
	for {
		if status, err := services.PingServices(); err != nil {
			log.Warnf("Error - ping_services: %v | status: %s\n", status, err)
			time.Sleep(time.Second)
			continue
		} else {
			log.Infof(">>>执行ping成功:%s", status) //绿色
		}
		for i, job := range models.JOBS {
			log.Printf(">>>执行任务%d %v\n", i, job)
			run_one(&job)
		}
		i++
		log.Infof(">>>执行次数：%d 完成", i)
		for ii := 0; ii < models.INTERVAL_SECONDS; ii++ {
			fmt.Printf(">>>等待%d秒，还需等待%d秒\r", ii, models.INTERVAL_SECONDS-ii)
			time.Sleep(time.Second)
		}

	}

}
