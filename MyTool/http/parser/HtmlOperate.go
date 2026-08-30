package parser

import (
	"fmt"
	"golang.org/x/net/html"
	"strings"
)

type MyHtml struct {
	doc *html.Node
}

func NewMyHtml(body string) *MyHtml {
	doc, _ := html.Parse(strings.NewReader(body))
	return &MyHtml{doc: doc}
}
func (this *MyHtml) Print() *MyHtml {
	fmt.Print(this.doc.Data)
	return this
}
