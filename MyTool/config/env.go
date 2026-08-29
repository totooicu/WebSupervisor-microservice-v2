package MyTool

import (
	"os"
	"regexp"
)

// ExpandEnvVars 替换字符串中的环境变量 ${ENV_NAME}
func ExpandEnvVars(s string) string {
	// 使用正则表达式匹配 ${ENV_NAME} 格式
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	
	return re.ReplaceAllStringFunc(s, func(match string) string {
		// 提取环境变量名（去掉 ${ 和 }）
		envName := match[2 : len(match)-1]
		
		// 获取环境变量值
		if value, exists := os.LookupEnv(envName); exists {
			return value
		}
		
		// 如果环境变量不存在，返回原始字符串
		return match
	})
}

// ExpandEnvVarsInMap 递归替换map中的环境变量
func ExpandEnvVarsInMap(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			m[k] = ExpandEnvVars(val)
		case map[string]interface{}:
			ExpandEnvVarsInMap(val)
		case []interface{}:
			ExpandEnvVarsInSlice(val)
		}
	}
}

// ExpandEnvVarsInSlice 递归替换slice中的环境变量
func ExpandEnvVarsInSlice(s []interface{}) {
	for i, v := range s {
		switch val := v.(type) {
		case string:
			s[i] = ExpandEnvVars(val)
		case map[string]interface{}:
			ExpandEnvVarsInMap(val)
		case []interface{}:
			ExpandEnvVarsInSlice(val)
		}
	}
}
