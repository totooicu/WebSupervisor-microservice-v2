package models
var EMAILS map[string]*EmailConfig
var DEFAULT_EMAIL string


type EmailConfig struct {
    Host               string `json:"host"`
    Port               int    `json:"port"`
    Username           string `json:"username"`
    Password           string `json:"password"`
    WaitTimeMS         int    `json:"wait_time_ms"`
}

type EmailContent struct{
	Tos       []string `json:"tos"`
    Subject  string `json:"subject"`
    Body     string `json:"body"`
}
type EmailRequestByConfig struct {//根据配置发送邮件
	EmailChoose string `json:"email_choose"`//空为default_email配置
	EmailContent EmailContent `json:"email_content"`
}
type EmailRequestByCustom struct {//根据自定义配置发送邮件
	EmailConfig EmailConfig `json:"email_config"`
    EmailContent EmailContent `json:"email_content"`
}


