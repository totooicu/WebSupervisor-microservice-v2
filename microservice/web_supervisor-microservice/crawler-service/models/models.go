package models

type CrawlerParameter struct {
	URL        string                 `json:"url"`
	Method     string                 `json:"method"`
	Headers    map[string]string      `json:"headers"`
	Body       map[string]interface{} `json:"body"`
	StrPayload string                 `json:"str_payload"`
}