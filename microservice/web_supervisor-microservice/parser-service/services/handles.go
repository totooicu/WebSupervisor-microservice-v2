package services

import (
	"github.com/totooicu/go-mytool/stream"
	"github.com/totooicu/parser-service/models"
	mystr "github.com/totooicu/go-mytool/string"
	log "github.com/sirupsen/logrus"
	"fmt"
	"encoding/json"
	"strings"
	"github.com/totooicu/go-mytool/http/parser"
)

func HandleParseHTMLByGetMid(msg *stream.StreamMessage) {
	// 解析HTML内容
	// 解析参数
	var params models.ParserParameter
	playloadData, err := json.Marshal(msg.Playload)
	if err != nil {
		log.Printf("Error marshalling playload: %v", err)
		return
	}
	
	if err := json.Unmarshal(playloadData, &params); err != nil {
		log.Printf("Error unmarshalling playload: %v", err)
		return
	}
	log.Printf(">>>Debug - HTML parse params: HTMLKeys=%v, content length=%d", params.HTMLKeys, len(params.Content))
	var results_mids map[string][]string
	results_mids=make(map[string][]string)
	var resultss map[string][]string
	resultss=make(map[string][]string)
	for i := 0; i < len(params.HTMLKeys); i++ {
		results_mids[params.HTMLKeys[i].Left]=mystr.GetMid (params.Content, params.HTMLKeys[i].Left, params.HTMLKeys[i].Right, 0)
	}
	for ii:=0;ii<len(params.HTMLKeys);ii++{
		results_mid:=results_mids[params.HTMLKeys[ii].Left]
		var results[]string

			//results[i]是否能与params.HTMLKeys[0].Keys匹配成功
	for i := 0; i < len(results_mid); i++ {
		// Index := stringtool.FindIndex(results_mid[i], ".*")
		// log.Printf("Debug - HTML parse Index : %v", Index)

		if mystr.StringMustCompileStringArray(results_mid[i], params.HTMLKeys[0].Keys) {
			results = append(results, results_mid[i])
		}
	}
	key:=fmt.Sprintf("[%s,%s,%s]",params.HTMLKeys[ii].Left,params.HTMLKeys[ii].Right,params.HTMLKeys[ii].Keys)
	resultss[key]=results
	}
	

	log.Printf("Debug - HTML parse completed, found %d results", len(resultss))
	paramData := map[string]interface{}{
		"parsed_data": resultss,
	}
	stream.ResponseSucc(msg,paramData)
}

func HandleParseHTMLByXPath(msg *stream.StreamMessage){
	var params models.ParserParameter
	playloadData, err := json.Marshal(msg.Playload)
	if err != nil {
		log.Printf("Error marshalling playload: %v", err)
		return
	}
	if err := json.Unmarshal(playloadData, &params); err != nil {
		log.Printf("Error unmarshalling playload: %v", err)
		return
	}
	
	XPaths:=params.XPaths
	content:=params.Content
	log.Printf(">>> %d %v",len(content),XPaths)
	var results map[string]([]string)
	results=make(map[string][]string)
	//results[i]是否能与params.HTMLKeys[0].Keys匹配成功
	for i := 0; i < len(XPaths); i++ {
		key:=XPaths[i].XPath
		results[key]=make([]string,0)
		if r,err:=parser.ParseHTMLByXpath(content,XPaths[i].XPath,XPaths[i].AttrName);err==nil {
			results[key] = append(results[key], r...)
		}else{
			log.Panic(err)
		}
	}

	log.Printf("Debug - HTML parse completed, found %d results", len(results))


	paramData := map[string]interface{}{
		"parsed_data": results,
	}
	stream.ResponseSucc(msg,paramData)
}


func  HandleParseJSON(msg *stream.StreamMessage) {
	// 解析参数
	var params models.ParserParameter
	playloadData, err := json.Marshal(msg.Playload)
	if err != nil {
		stream.ResponseErr(msg,err.Error())
		log.Printf("Error marshalling playload: %v", err)
		return
	}
	
	if err := json.Unmarshal(playloadData, &params); err != nil {
		stream.ResponseErr(msg,err.Error())
		log.Printf("Error unmarshalling playload: %v", err)
		return
	}

	// 将JSONKeys转换为utils.ParseJSON期望的格式 ["path1.path2", ...]
	var jsonKeys []string
	for _, jsonKey := range params.JSONKeys {
		var pathParts []string
		for _, pathPart := range jsonKey.Path {
			pathParts = append(pathParts, fmt.Sprintf("%v", pathPart))
		}
		pathStr := strings.Join(pathParts, ".")
		jsonKeys = append(jsonKeys, pathStr)
	}

		log.Printf("Debug - JSON parse params: JSONKeys=%v, content length=%d", params.JSONKeys, len(params.Content))

	results := parser.ParseJSON(params.Content, jsonKeys)

		log.Printf("Debug - JSON parse completed, found %d results", len(results))

	paramData := map[string]interface{}{
		"parsed_data": results,
	}

	stream.ResponseSucc(msg,paramData)

}
