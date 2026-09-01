package string

import (
	"regexp"
"unicode/utf8"
    "bytes"
    "fmt"
    "io/ioutil"
"golang.org/x/text/encoding/simplifiedchinese"
 "github.com/saintfish/chardet"
//    "golang.org/x/text/encoding"
    "golang.org/x/text/encoding/htmlindex"
    "golang.org/x/text/transform"
)

type MyString struct {
	str string
}

type FindIndexResult struct {
	str   string
	index int
}

// 正则表达式匹配位置
func FindIndex(str string, rsub string) []FindIndexResult {
	var res []FindIndexResult
	for _, match := range regexp.MustCompile(rsub).FindAllStringIndex(str, -1) {
		res = append(res, FindIndexResult{str: str[match[0]:match[1]], index: match[0]})
	}
	return res
}

// s是否在rsub中（正则）
func StringMustCompileStringArray(s string, rsub []string) bool {
	//var debug []string

	for _, v := range rsub {
		//debug = append(debug, fmt.Sprintf("[%s::%s]", s, v))
		if len(FindIndex(s, v)) != 0 {
			return true
		}
	}
	//cmd.Print(">>>StringMustCompileStringArray debug", debug)
	return false
}

// ss是否存在一个元素在rsub中（正则）
func StringArrayMustCompileStringArray(ss []string, rsub []string) bool {
	// fmt.Printf("StringArrayMustCompileStringArray %d*%d", len(ss), len(rsub))
	for _, v := range ss {
		if StringMustCompileStringArray(v, rsub) {
			return true
		}
	}
	return false
}

// 利用正则获取中间字符
// src=源字符串,rlstr=左边字符,rrstr=右边字符
// op=ab,a表示是否保留左边,b表示是否保留右边

func GetMid(src, rlstr, rrstr string, op byte) []string {
	var res []string
	if len(src) == 0 {
		return res
	}
	LP, RP := FindIndex(src, rlstr), FindIndex(src, rrstr)
	// fmt.Printf("GetMid  rlstr: %s, rrstr: %s\n",  rlstr, rrstr)
	// fmt.Printf("GetMid LP: %v, RP: %v\n", LP, RP)
		
	// 双指针匹配算法
	lpIndex := 0
	for _, vr := range RP {
		// 寻找最大的左边界（最后一个小于右边界位置的）
		lastValid := -1
		for lpIndex < len(LP) && LP[lpIndex].index < vr.index {
			lastValid = lpIndex
			lpIndex++
		}

		if lastValid == -1 {
			continue
		}

		// 计算内容范围
		vl := LP[lastValid]
		start := vl.index
		end := vr.index + len(vr.str)

		// 处理保留选项
		if op&1 == 0 { // 不保留左边界
			start += len(vl.str)
		}
		if op&2 == 0 { // 不保留右边界
			end = vr.index
		}

		// 获取匹配内容
		if start <= end && end <= len(src) {
			res = append(res, src[start:end])
		}

		// 跳过已处理的左边界
		lpIndex = lastValid + 1
	}
	return res
}

// ConvertToUTF8 自动检测字符集并转换为 UTF-8
func ConvertToUTF8(data []byte) ([]byte, error) {
    // 1. 快速检测是否为合法 UTF-8（同时包含纯 ASCII）
    if utf8.Valid(data) {
        return data, nil
    }

    // 2. 使用 chardet 检测字符集
    detector := chardet.NewTextDetector()
    result, err := detector.DetectBest(data)
    if err != nil {
        // 检测失败，尝试使用 HTML 标准字符集检测（通过 BOM 或 meta）
        // 这里简单回退：假设为 GBK（可根据业务调整）
        result = &chardet.Result{Charset: "GB18030"}
    }

    // 3. 根据检测到的字符集获取 encoding.Encoding
    enc, err := htmlindex.Get(result.Charset)
    if err != nil {
        // 尝试使用其他常见编码名称
        enc = simplifiedchinese.GBK // 回退到 GBK
    }

    // 4. 转换为 UTF-8
    reader := transform.NewReader(bytes.NewReader(data), enc.NewDecoder())
    utf8Bytes, err := ioutil.ReadAll(reader)
    if err != nil {
        return nil, fmt.Errorf("转换失败: %v", err)
    }
    return utf8Bytes, nil
}