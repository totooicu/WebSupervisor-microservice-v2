package MyTool

import (
	"fmt"
	"io/ioutil"
	"os"
)

// 读取到file中，再利用ioutil将file直接读取到[]byte中, 这是最优
func Read(fileName string) string {
	f, err := os.Open(fileName)
	if err != nil {
		fmt.Print("read file fail", err)
		return ""
	}
	defer f.Close()
	fd, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Print("read to fd fail", err)
		return ""
	}
	return string(fd)
}
func Write(fileName string, data string) {
	err := ioutil.WriteFile(fileName, []byte(data), 0666)
	if err != nil {
		fmt.Print("write fail ", err)
	}
	fmt.Print("write success")
}
func CheckFileExist(fileName string) bool {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}
	return true
}
