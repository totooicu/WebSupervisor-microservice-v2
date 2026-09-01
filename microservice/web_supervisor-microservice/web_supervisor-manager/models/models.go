package models
import (
	
)
type EmailContent struct{
	Tos       []string `json:"tos"`
    Subject  string `json:"subject"`
    Body     string `json:"body"`
}
type EmailRequestByConfig struct {//根据配置发送邮件
	EmailChoose string `json:"email_choose"`//空为default_email配置
	EmailContent EmailContent `json:"email_content"`
}


//爬取-Keys->解析-Cache->缓存
type Job struct {
	JobType  string `json:"job_type"`
	Crawler CrawlerParameter `json:"crawle"`
	Command CommandParameter `json:"command"`
	Keys    ParserParameter    `json:"keys"`
	Caches   []CacheParameter      `json:"caches"`
}
type CommandParameter struct { 
	Command string `json:"command"`
	Dir string `json:"dir"`
	TimeoutMs int `json:"timeout_ms"`
}

// CrawlerParameter 爬虫服务参数结构体
type CrawlerParameter struct {
	URL        string                 `json:"url"`
	Method     string                 `json:"method"`
	Header    map[string]string      `json:"header"`
	Body       map[string]any `json:"body"`
	StrPayload string                 `json:"str_payload"`
}

// ParserParameter 解析服务参数结构体
type ParserParameter struct {
	Content  string                 `json:"content"`
	HTMLKeys []map[string]any `json:"htmlKeys"`
	JSONKeys []map[string]any `json:"jsonKeys"`
	XPathKeys []map[string]any `json:"xpathKeys"`
}

// CacheParameter 缓存服务参数结构体
type CacheParameter struct {
	App  string      `json:"app"`
	Key  string      `json:"key"`
	Data any `json:"data"`
}

// MonitorParameter 监控服务参数结构体
type MonitorParameter struct {
	JobID         string           `json:"job_id"`
	Interval      int              `json:"interval"`
	CrawlerParams CrawlerParameter `json:"crawler_params"`
}