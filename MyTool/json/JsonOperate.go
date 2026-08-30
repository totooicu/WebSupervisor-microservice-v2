package json

import (
	"encoding/json"
)



// func MapToStruct(data map[string]interface{}, s any) error {
// 	return json.Unmarshal(MapToJsonBytes(data), s)
// }
// MapToJsonBytes 将 map 转为 JSON 字节，便于反序列化
func MapToJsonBytes(data map[string]interface{}) []byte {
	b, _ := json.Marshal(data)
	return b
}

func MapToJsonToStruct(data map[string]interface{}, s any) error {
	return json.Unmarshal(MapToJsonBytes(data), s)
}
func AnyToStruct(data any, s any) error {
	return json.Unmarshal(MapToJsonBytes(data.(map[string]interface{})), s)
}
func StructToMap(s any) map[string]interface{} {
	var r map[string]interface{}
	b, _ := json.Marshal(s)
	json.Unmarshal(b, &r)
	return r
}

func StringToMap(jsonstr string) map[string]interface{} {
	var r map[string]interface{}
	json.Unmarshal([]byte(jsonstr), &r)
	return r
}
func AnyToMap(data any) map[string]any {
	return data.(map[string]any)
}
