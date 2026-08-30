package config

import (
	"encoding/json"
	"io/ioutil"
)

// MapToStruct 将map转换为结构体
func MapToStruct(m map[string]interface{}, s interface{}) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, s)
}

// LoadConfig 加载配置文件并解析为指定结构体
func LoadConfig(configPath string, config interface{}) error {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	// 先解析为map，应用环境变量替换
	var configMap map[string]interface{}
	if err := json.Unmarshal(data, &configMap); err != nil {
		return err
	}

	// 替换环境变量
	ExpandEnvVarsInMap(configMap)

	// 将map转换回JSON
	processedData, err := json.Marshal(configMap)
	if err != nil {
		return err
	}

	// 解析为配置结构体
	return json.Unmarshal(processedData, config)
}