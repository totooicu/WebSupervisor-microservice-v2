package models

import (
	"github.com/totooicu/go-mytool/json"
	"errors"
)

var (
	CRAWLER_SERVICE=""
	EMAIL_SERVICE=""
	PARSER_SERVICE=""
	REDIS_CACHE_SERVICE=""
	REDIS_APP_NAME string=""
	EMAIL_TOS []string = []string{}          // 监控变化通知收件人（仅消息）
	EMAIL_TOS_ADMINS []string = []string{}   // 管理员通知收件人（错误/告警等其它信息）
	INTERVAL_SECONDS=3600
	
	JOBS_NAVIGATOR *json.JSONNavigator
	JOBS []Job

)
var ErrCommandTimeout = errors.New("command execution timed out")