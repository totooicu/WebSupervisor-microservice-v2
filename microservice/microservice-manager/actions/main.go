package actions

import("log")
func testEMail(){
	sendEmailByConfig()
	sendEmailByCustom()
}
func testRedisCache(){
	set("app","key","value3")
	get("app","key")
	set("app","key","value1")
	compareAndSave("app","key","value1")
	compareAndSave("app","key","value2")
	compareAndSave("app","key","value2")
	get("app","key")
	
	get("app","key")
	// delete("app","key")
	getAndSet("app","key","value4")
	get("app","key")
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
	testRedisCache()
	// testHttpRequestHttpRequest()
	// testParser()
	// testBasic()
}