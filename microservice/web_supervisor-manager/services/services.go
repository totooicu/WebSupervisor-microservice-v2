package services

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/totooicu/go-mytool/json"
	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/web_supervisor-manager/models"
)

func Crawl(p *models.CrawlerParameter) (map[string]any, error) {
	streamMsg, err := stream.Send(models.CRAWLER_SERVICE, "http_request", json.StructToMap(p), 1000*60*2) //->{"content":"","status":0}
	if err != nil {
		log.Warnf("Crawl Error - Send message to stream: err:%v | msg:%v", err, streamMsg.ErrMsg)
		return nil, err
	}
	return streamMsg.Playload, nil
}
func Parse(p *models.ParserParameter) (map[string]any, error) {
	log.Printf(">>> Debug - p: v %v %v %v", p.HTMLKeys, p.XPathKeys, p.JSONKeys)
	serv := ""
	if p.HTMLKeys != nil && len(p.HTMLKeys) != 0 {
		serv = "parse_html_by_get_mid"
	}
	if p.XPathKeys != nil && len(p.XPathKeys) != 0 {
		serv = "parse_html_by_xpath"
	}
	if p.JSONKeys != nil && len(p.JSONKeys) != 0 {
		serv = "parse_json"
	}
	// log.Printf(">>> Debug - serv: %s", serv)
	// return nil, fmt.Errorf("serv is empty")
	if serv == "" {
		serv = "parse_html_by_get_mid"
		p.HTMLKeys = make([]map[string]any, 1)
		p.HTMLKeys[0] = map[string]any{
			"left":  "<html>",
			"right": "</html>",
		}
		p.HTMLKeys[0]["keys"] = []string{".*"}
	}
	log.Printf(">>> Debug - serv: %s", serv)

	streamMsg, err := stream.Send(models.PARSER_SERVICE, serv, json.StructToMap(p), 1000*10)
	//parse_html_by_xpath->{"parsed_data":map[string]([]string)}, parsed_data[xpath]=[content]
	//parse_html_by_get_mid->{"parsed_data":map[string]([]string)[]string}, parsed_data[key]=[content], key:=fmt.Sprintf("[%s,%s,%s]",params.HTMLKeys[ii].Left,params.HTMLKeys[ii].Right,params.HTMLKeys[ii].Keys)
	//parse_json->{"parsed_data":map[string]([]any)}, parsed_data[jsonpath]=[value]

	if err != nil {
		log.Warnf("Error - Send message to stream: %v", err)
		return nil, err
	}
	return streamMsg.Playload, nil
}
func CacheCompareAndSave(p *models.CacheParameter) (map[string]any, error) {
	streamMsg, err := stream.Send(models.REDIS_CACHE_SERVICE, "compare_and_save", json.StructToMap(p), 1000*10) //->{"changed":true}
	if err != nil {
		log.Warnf("Error - Send message to stream: %v", err)
		return nil, err
	}
	if streamMsg.ErrMsg != "" {
		log.Warnf("Error - CacheCompareAndSave: %v", streamMsg.ErrMsg)
		return nil, fmt.Errorf("%v", streamMsg.ErrMsg)
	}
	return streamMsg.Playload, nil
}
func CacheSet(p *models.CacheParameter) (map[string]any, error) {
	streamMsg, err := stream.Send(models.REDIS_CACHE_SERVICE, "set", json.StructToMap(p), 1000*10) //->{"key":"",error:""}
	if err != nil {
		log.Warnf("Error - Send message to stream: %v", err)
		return nil, err
	}
	if streamMsg.ErrMsg != "" {
		log.Warnf("Error - CacheSet: %v", streamMsg.ErrMsg)
		return nil, fmt.Errorf("%v", streamMsg.ErrMsg)
	}
	return streamMsg.Playload, nil
}
func EmailTemplate(kv map[string]any) string {
	if len(kv) == 0 {
		return ""
	}
	return kv["body"].(string)
}
func EmailNoticeChanged(p *[]*models.CacheParameter, url *string) (map[string]any, error) {

	title := fmt.Sprintf("监听到变化变化的链接:%s\n内容", *url)
	content := ""

	for i, v := range *p {
		c:= fmt.Sprintf(">>>%d\t%s:%s\n", i, v.Key, v.Data)
		c=c[:min(len(c), 1000)]
		content += c
		}
	content=content[:(min(len(content), 100000))]
	sender := models.EmailRequestByConfig{
		EmailChoose: "",
		EmailContent: models.EmailContent{
			Tos:     models.EMAIL_TOS,
			Subject: title,
			Body:    content,
		},
	}
	streamMsg, err := stream.Send(models.EMAIL_SERVICE, "email_by_config", json.StructToMap(sender), 1000*10) //->{nil}
	if err != nil {
		log.Warnf("Error - Send message to stream: %v", err)
		return nil, err
	}
	return streamMsg.Playload, nil
}
