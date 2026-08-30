package parser

import (
	"strings"
	"fmt"
	"github.com/antchfx/htmlquery"
)

// ExtractByXPath 从 HTML 文档中根据 XPath 提取文本内容
// htmlContent: HTML 文档字符串
// xpathExpr: XPath 表达式，如 "//div[@class='content']/p"
// 返回所有匹配节点的 InnerText 切片，若没有匹配返回空切片
func ExtractByXPath(htmlContent, xpathExpr string) ([]string, error) {
	// 解析 HTML
	doc, err := htmlquery.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	// 执行 XPath 查询
	nodes, err := htmlquery.QueryAll(doc, xpathExpr)
	if err != nil {
		return nil, fmt.Errorf("xpath query: %w", err)
	}

	results := make([]string, 0, len(nodes))
	for _, node := range nodes {
		text := htmlquery.InnerText(node) // 获取节点内所有文本
		results = append(results, text)
	}
	return results, nil
}

// 示例：提取属性值（如链接地址）
func ExtractAttrByXPath(htmlContent, xpathExpr, attrName string) ([]string, error) {
	doc, err := htmlquery.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, err
	}
	nodes, err := htmlquery.QueryAll(doc, xpathExpr)
	if err != nil {
		return nil, err
	}
	var attrs []string
	for _, node := range nodes {
		attr := htmlquery.SelectAttr(node, attrName)
		attrs = append(attrs, attr)
	}
	return attrs, nil
}

func ParseHTMLByXpath(content string, xpath string,attrName string) ([]string,error ){
	if attrName == "" {
		return ExtractByXPath(content, xpath)
	}
	return ExtractAttrByXPath(content, xpath, attrName)
}