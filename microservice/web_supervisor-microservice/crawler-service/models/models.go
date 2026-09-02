package models

type CrawlerParameter struct {
	URL        string                 `json:"url"`
	Method     string                 `json:"method"`
	Headers    map[string]any      `json:"headers"`
	Body       map[string]any      `json:"body"`
	StrPayload string                 `json:"str_payload"`
}