package json

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type MyJson struct {
	j     map[string]any
	curj  map[string]any
	cura  []any
	Res   any
	curjs []map[string]any
	Ress  []any
}

func CanConvert(jsonstr string) bool {
	js := MyJson{}
	e := json.Unmarshal([]byte(jsonstr), &js.j)
	return e == nil
}

func NewJson(jsonstr string) *MyJson {
	js := MyJson{}
	json.Unmarshal([]byte(jsonstr), &js.j)
	js.curj = js.j
	return &js
}
func NewJsons(jsonstr string) *MyJson {
	js := MyJson{}
	json.Unmarshal([]byte(jsonstr), &js.j)
	js.cura = append(js.cura, js.j)
	return &js
}
func (this *MyJson) GetCur(key any) any {
	if reflect.TypeOf(key).Kind() == reflect.String {
		return this.curj[key.(string)]
	} else {
		return this.cura[int(key.(float64))]
	}
}
func (this *MyJson) Get(path []any) *MyJson {
	//this.print()
	if len(path) == 1 {
		this.Res = this.GetCur(path[0])
	} else {
		//if path[1] is string then save to curj and let cura=nil;
		if reflect.TypeOf(path[1]).Kind() == reflect.String {
			this.curj = this.GetCur(path[0]).(map[string]any)
			this.cura = nil
		} else {
			this.cura = this.GetCur(path[0]).([]any)
			this.curj = nil
		}
		this.Get(path[1:])
	}
	return this
}

// 针对path=-1，获取所有
// key=-1，获取所有，key为string则返回全部数组并get map的值，key为非负整数返回数组第key个
func (this *MyJson) GetCurs(key any) []any {
	var ans []any
	if reflect.TypeOf(key).Kind() == reflect.String {
		for _, v := range this.cura {
			ans = append(ans, v.(map[string]any)[key.(string)])
		}
		return ans
	} else {
		if int(key.(float64)) != -1 {
			for _, v := range this.cura {
				ans = append(ans, v.([]any)[int(key.(float64))])
			}
			return ans
		} else {
			for _, v := range this.cura {
				for _, vv := range v.([]any) {
					ans = append(ans, vv)
				}
			}
			return ans
		}
	}
}

// 结果都存入cura中
func (this *MyJson) Gets(path []any) *MyJson {
	if len(path) == 1 {
		this.Ress = this.GetCurs(path[0])
	} else {
		this.cura = this.GetCurs(path[0])
		this.Gets(path[1:])
	}
	return this
}
func (this *MyJson) print() {
	fmt.Printf(">>>[curj=%v,cura=%v]\n", this.curj, this.cura)
}

// MapToJsonBytes 将 map 转为 JSON 字节，便于反序列化
func MapToJsonBytes(data map[string]interface{}) []byte {
	b, _ := json.Marshal(data)
	return b
}