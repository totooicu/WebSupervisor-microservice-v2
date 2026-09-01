package actions

import (
	"log"

	"github.com/sirupsen/logrus"
	"github.com/totooicu/go-mytool/file"
	"github.com/totooicu/go-mytool/json"
	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/go-mytool/http/parser"
)

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
	XPaths   []XPathKey  `json:"xPaths"`
}



func readHtml()string{
	return file.Read("resource\\html1.html")
}
func qByGetMid(){

}
func qStr(s string){
	res,err:= parser.ExtractByXPath(s,"/html/body/div[5]/div[2]/div[1]/div[4]/ul/li")
	if err!=nil{
		log.Println(err)
	}
	log.Println(res)
}
func qByXPath(s string){
	pp:=ParserParameter{
		Content: s,
		XPaths: []XPathKey{{"/html/body/div[5]/div[2]/div[1]/div[4]/ul/li",""}},
	}
	res,err:=stream.Send("parser-stream","parse_html_by_xpath",json.StructToMap(pp),0)
	if err!=nil{
		log.Println(err)
	}
	logrus.Print(res.Playload)
}