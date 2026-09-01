package services

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/totooicu/go-mytool/encryption"
	"github.com/totooicu/go-mytool/json"
	"github.com/totooicu/go-mytool/stream"
	myString "github.com/totooicu/go-mytool/string"
	"os/exec"
	"bytes"
	"errors"
	"context"
	"github.com/totooicu/web_supervisor-manager/models"
)

// 错误去重缓存有效期：同一签名在此期间重复出现只向管理员告警一次
var errorDedupTTL = 24 * time.Hour

func RunCommand(p *models.CommandParameter) (string, error) {
    var cmd *exec.Cmd
    var cancel context.CancelFunc

    if p.TimeoutMs > 0 {
        timeout := time.Duration(p.TimeoutMs) * time.Millisecond
        ctx, c := context.WithTimeout(context.Background(), timeout)
        cancel = c
        defer cancel()
        cmd = exec.CommandContext(ctx, "cmd","/C",p.Command)
    } else {
        cmd = exec.Command("cmd","/C",p.Command)
    }

    cmd.Dir = p.Dir

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err := cmd.Run()
	d,_:=myString.ConvertToUTF8(stdout.Bytes())
	log.Printf(">>> Debug - RunCommand utf8: %s", string(d))
	stderr_utf8,_:=myString.ConvertToUTF8(stderr.Bytes())

    if err != nil {
		err_utf8,_:=myString.ConvertToUTF8([]byte(fmt.Sprintf("%v",err)))
		log.Printf(">>> Debug - RunCommand utf8 stderr: %s", string(stderr_utf8))
		log.Printf(">>> Debug - RunCommand utf8 err: %s", string(err_utf8))
        // 检查是否是超时错误
        if errors.Is(err, context.DeadlineExceeded) {
            return stdout.String(), fmt.Errorf("%w: %v", models.ErrCommandTimeout, err)
        }
        // 其他错误，附加上 stderr 内容
        if stderr.Len() > 0 {
            return stdout.String(), fmt.Errorf("%v: %s", err, string(stderr_utf8))
        }
        return stdout.String(), err
    }

    return stdout.String(), nil
}
func Crawl(p *models.CrawlerParameter) (map[string]any, error) {
	log.Printf(">>> Debug - Send http_request")
	streamMsg, err := stream.Send(models.CRAWLER_SERVICE, "http_request", json.StructToMap(p), 1000*60*2) //->{"content":"","status":0}
	log.Printf("<<< Debug - Send http_request ok")
	if err != nil {
		// if streamMsg.ErrMsg =="" {
		// 	return nil, fmt.Errorf("Error - Send message to stream: %v", err)
		// }
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
		log.Printf(">>> Debug - p.HTMLKeys: %v", p.HTMLKeys)
	}
	if p.XPathKeys != nil && len(p.XPathKeys) != 0 {
		serv = "parse_html_by_xpath"
		log.Printf(">>> Debug - p.XPathKeys: %v", p.XPathKeys)
	}
	if p.JSONKeys != nil && len(p.JSONKeys) != 0 {
		serv = "parse_json"
		log.Printf(">>> Debug - p.JSONKeys: %v", p.JSONKeys)
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
	log.Printf(">>> Debug - send parse request")
	streamMsg, err := stream.Send(models.PARSER_SERVICE, serv, json.StructToMap(p), 1000*10)
	//parse_html_by_xpath->{"parsed_data":map[string]([]string)}, parsed_data[xpath]=[content]
	//parse_html_by_get_mid->{"parsed_data":map[string]([]string)[]string}, parsed_data[key]=[content], key:=fmt.Sprintf("[%s,%s,%s]",params.HTMLKeys[ii].Left,params.HTMLKeys[ii].Right,params.HTMLKeys[ii].Keys)
	//parse_json->{"parsed_data":map[string]([]any)}, parsed_data[jsonpath]=[value]
	log.Printf("<<< Debug - send parse request ok")
	if err != nil {
		log.Warnf("Error - Send message to stream: %v", err)
		return nil, err
	}
	return streamMsg.Playload, nil
}
func CacheCompareAndSave(p *models.CacheParameter) (map[string]any, error) {
	log.Printf(">>> Debug - send CacheCompareAndSave")
	streamMsg, err := stream.Send(models.REDIS_CACHE_SERVICE, "compare_and_save", json.StructToMap(p), 1000*10) //->{"changed":true}
	log.Printf("<<< Debug - CacheCompareAndSave ok")
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
	log.Printf(">>> Debug - send CacheSet")
	streamMsg, err := stream.Send(models.REDIS_CACHE_SERVICE, "set", json.StructToMap(p), 1000*10) //->{"key":"",error:""}
	log.Printf("<<< Debug - CacheSet ok")
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
		c := fmt.Sprintf(">>>%d\t%s:%s\n", i, v.Key, v.Data)
		c = c[:min(len(c), 1000)]
		content += c
	}
	content = content[:(min(len(content), 100000))]
	sender := models.EmailRequestByConfig{
		EmailChoose: "",
		EmailContent: models.EmailContent{
			Tos:     models.EMAIL_TOS,
			Subject: title,
			Body:    content,
		},
	}
	log.Printf(">>> Debug - send email_by_config")
	streamMsg, err := stream.Send(models.EMAIL_SERVICE, "email_by_config", json.StructToMap(sender), 1000*10) //->{nil}
	log.Printf("<<< Debug - send email_by_config ok")
	if err != nil {
		log.Warnf("Error - Send message to stream: %v", err)
		return nil, err
	}
	return streamMsg.Playload, nil
}

func PingServices() (string, error) {
	status := ""
	if s, err := stream.SendPing(models.CRAWLER_SERVICE); err != nil {
		status += fmt.Sprintf("Crawler_service:%s\n", s.Playload)
		return status, err
	}
	if s, err := stream.SendPing(models.PARSER_SERVICE); err != nil {
		status += fmt.Sprintf("Parser_service:%s\n", s.Playload)
		return status, err
	}
	if s, err := stream.SendPing(models.REDIS_CACHE_SERVICE); err != nil {
		status += fmt.Sprintf("Redis_cache_service:%s\n", s.Playload)
		return status, err
	}
	if s, err := stream.SendPing(models.EMAIL_SERVICE); err != nil {
		status += fmt.Sprintf("Email_service:%s\n", s.Playload)
		return status, err
	}
	return status, nil
}

// ReportError 缓存错误到 Redis 并按需通知管理员。
// 相同签名（url+stage+msg）在 errorDedupTTL 内重复出现时只缓存、不再发邮件，
// 避免重复错误刷屏管理员。首次出现的错误才会发送告警邮件。
func ReportError(jobURL, stage, msg string) {
	signature := fmt.Sprintf("%s|%s|%s", jobURL, stage, msg)
	key := models.REDIS_APP_NAME + ":err_dedup:" + encryption.HashMD5(signature)
	rc := stream.GetMyRedisClient()

	var existing string
	if err := rc.GetKey(key, &existing); err == nil {
		// 已缓存过该错误：重复，不发邮件
		log.Warnf("Error (suppressed, already reported): %s", signature)
		return
	}
	// 新错误：先缓存再通知
	if err := rc.SetKey(key, signature, errorDedupTTL); err != nil {
		log.Warnf("Error - cache error signature: %v", err)
	}
	sendAdminNotice(
		fmt.Sprintf("[web_supervisor] %s @ %s", stage, jobURL),
		fmt.Sprintf("URL: %s\nstage: %s\nerror: %s\n\n(相同错误将不会重复告警)", jobURL, stage, msg),
	)
}

// sendAdminNotice 向管理员收件人发送告警邮件。未配置 email_tos_admin 时仅记录日志。
func sendAdminNotice(subject, body string) {
	if len(models.EMAIL_TOS_ADMINS) == 0 {
		log.Warnf("[admin] no admin recipients configured, skip notice: %s", subject)
		return
	}
	sender := models.EmailRequestByConfig{
		EmailChoose: "",
		EmailContent: models.EmailContent{
			Tos:     models.EMAIL_TOS_ADMINS,
			Subject: subject,
			Body:    body,
		},
	}
	log.Printf(">>> Debug - send admin notice: %s", subject)
	streamMsg, err := stream.Send(models.EMAIL_SERVICE, "email_by_config", json.StructToMap(sender), 1000*10)
	if err != nil {
		msgStr := ""
		if streamMsg != nil {
			msgStr = streamMsg.ErrMsg
		}
		log.Warnf("Error - send admin notice: %v | msg: %v", err, msgStr)
	}
}
