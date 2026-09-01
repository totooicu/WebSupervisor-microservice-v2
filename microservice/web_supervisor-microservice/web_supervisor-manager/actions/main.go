package actions

import("log"

)


func testEMail(){
	sendEmailByConfig()
	sendEmailByCustom()
}
func testRedisCache(){
	set("app","key1","value1");get("app","key1")
	compareAndSave("app","key1","value2")

	set("app","key1",[]any{"value1","value3"});get("app","key1")
	compareAndSave("app","key1",[]any{"value1","value2"})

	set("app","key1",map[string]any{"key1":"value1","key2":"value4"});get("app","key1")
	compareAndSave("app","key1",map[string]any{"key1":"value1","key2":"value2"})
}

func testHttpRequestHttpRequest(){
	HttpRequestHttpRequest(&cfg1)
}
func testParser(){
	s:=readHtml()
	qByXPath(s)
	log.Printf("s.len:%v\n",len(s))
	// qStr(s)
}
func testBasic(){
	ping()
}


func Action(){
	// testEMail()
	// testRedisCache()
	// testHttpRequestHttpRequest()
	// testParser()
	// testBasic()

	runs()
}