package http

import (
	myjson "github.com/totooicu/go-mytool/json"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type HttpHeader struct {
	Url        string
	Header     map[string]string
	response   *http.Response
	bodyString string
	statusCode int
}

func NewHttpHeader(url string, header map[string]string) *HttpHeader {
	return &HttpHeader{Url: url, Header: header}
}
func (this *HttpHeader) Get(param string) *HttpHeader {
	client := &http.Client{}
	req, err := http.NewRequest("GET", this.Url+"?"+param, nil)
	
	if err != nil {
		// 处理错误
		fmt.Print("HttpClient Get req 失败", err)
	}
	for k, v := range this.Header {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		// 处理错误
		fmt.Print("HttpClient Get resp 请求失败", err.Error())
	}
	// 读取响应体
	this.response = resp
	return this
}
func (this *HttpHeader) Post(data map[string]any, stringPlayLoad string) *HttpHeader {
	client := &http.Client{}

	jsonData := stringPlayLoad
	// 发送JSON数据
	if stringPlayLoad == "" {
		jsonBytsData, _ := json.Marshal(data)
		jsonData = string(jsonBytsData)
	}

	//jsonData := `{"name": "John Doe"}`
	req, err := http.NewRequest("POST", this.Url, strings.NewReader(jsonData))
	//req.Header.Add("Content-Type", "application/json")
	fmt.Print(">>>HttpClient [url,jsonData]", this.Url, jsonData)
	for k, v := range this.Header {
		//fmt.Print(">>>HttpClient header:", k, v)
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		// 处理错误
		fmt.Print("HttpClient Post 请求失败", err)
	} else {
		fmt.Print(">>>HttpClient Post 响应状态码:", resp.StatusCode)
	}
	this.response = resp

	return this
}
func (this *HttpHeader) Read() *HttpHeader {
	this.statusCode = this.response.StatusCode
	if this.response.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(this.response.Body)
		if err != nil {
			// 处理错误
			fmt.Print("HttpClient Read bodyBytes 读取失败", err)
		}
		this.bodyString = string(bodyBytes)
		// 使用响应内容
	} else {
		fmt.Print("HttpClient Read StatusCode 错误", this.response.StatusCode)
	}
	return this
}
func (this *HttpHeader) GetBodyString() string {
	//fmt.Print("HttpClient GetBodyString", this.bodyString)
	return this.bodyString
}
func (this *HttpHeader) GetStatusCode() int { 
	return this.statusCode
}
func (this *HttpHeader) Close() {
	err := this.response.Body.Close()
	if err != nil {
		return
	}
}
func (this *HttpHeader) JsonBodyOperate() myjson.JSONNavigator {
	return myjson.JSONNavigator{}
}
