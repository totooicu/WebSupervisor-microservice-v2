package models

// HTMLKey HTML解析键结构体
type HTMLKey struct {
	Left  string   `json:"left"`
	Right string   `json:"right"`
	Keys  []string `json:"key"`
}

// JSONKey JSON解析键结构体
type JSONKey struct {
	Path []interface{} `json:"path"`
	Keys []string      `json:"key"`
}
type XPathKey struct {
	XPath string `json:"xpath"`
	AttrName string `json:"attrName"`
}

// ParserParameter 解析服务参数结构体
type ParserParameter struct {
	Content  string    `json:"content"`
	HTMLKeys []HTMLKey `json:"htmlKeys"`
	JSONKeys []JSONKey `json:"jsonKeys"`
	XPaths   []XPathKey  `json:"xpathKeys"`
}
