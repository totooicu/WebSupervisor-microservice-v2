package models

import (
	"github.com/totooicu/go-mytool/json"
)

var (
	CRAWLER_SERVICE=""
	EMAIL_SERVICE=""
	PARSER_SERVICE=""
	REDIS_CACHE_SERVICE=""
	REDIS_APP_NAME string=""
	EMAIL_TOS []string = []string{}
	INTERVAL_SECONDS=3600
	
	JOBS_NAVIGATOR *json.JSONNavigator
	JOBS []Job

)