package json

import (
	"encoding/json"
	"fmt"
)

// JSONNavigator 提供对 JSON 数据的路径导航和提取功能。
// 支持链式调用，每一步的错误会累积在内部，后续调用自动跳过，直到通过 Error() 获取。
type JSONNavigator struct {
	root        any   // 解析后的完整 JSON 数据（map[string]any 或 []any）
	current     any   // 当前导航位置
	lastResult  any   // 最近一次 Get 的结果
	lastResults []any // 最近一次 Gets 的结果
	err         error // 累积的错误（第一个错误会被保留）
}

// CanConvert 检查字符串是否为合法的 JSON 格式。
func CanConvert(jsonStr string) bool {
	var temp any
	err := json.Unmarshal([]byte(jsonStr), &temp)
	return err == nil
}

// NewJSONNavigator 从 JSON 字符串创建导航器。
// 解析失败时返回 nil 和错误（此处错误不会累积到链式调用中）。
func NewJSONNavigator(jsonStr string) (*JSONNavigator) {
	var data any
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return &JSONNavigator{
			err: fmt.Errorf("解析 JSON 失败: %w", err),
		}
	}
	return &JSONNavigator{
		root:    data,
		current: data,
	}
}

// Error 返回链式调用过程中累积的第一个错误。
func (n *JSONNavigator) Error() error {
	return n.err
}

// setError 设置错误（仅当尚未有错误时）。
func (n *JSONNavigator) setError(err error) {
	if n.err == nil && err != nil {
		n.err = err
	}
}

// Get 根据路径（例如 []any{"a", 0, "b"}）导航到指定位置。
// 返回 *JSONNavigator 支持链式调用，错误通过 Error() 获取。
func (n *JSONNavigator) Get(path []any) *JSONNavigator {
	if n.err != nil {
		return n // 已有错误，跳过执行
	}
	if len(path) == 0 {
		n.setError(fmt.Errorf("路径不能为空"))
		return n
	}

	// 从根开始
	n.current = n.root

	for _, key := range path {
		val, err := n.getCurrentValue(key)
		if err != nil {
			n.setError(err)
			return n
		}
		n.current = val
	}
	n.lastResult = n.current
	return n
}

// getCurrentValue 从当前导航位置获取 key 对应的值（内部方法）。
func (n *JSONNavigator) getCurrentValue(key any) (any, error) {
	switch k := key.(type) {
	case string:
		obj, ok := n.current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("当前不是对象，无法用字符串键 %q 访问", k)
		}
		val, exists := obj[k]
		if !exists {
			return nil, fmt.Errorf("键 %q 不存在", k)
		}
		return val, nil
	case int:
		arr, ok := n.current.([]any)
		if !ok {
			return nil, fmt.Errorf("当前不是数组，无法用索引 %d 访问", k)
		}
		if k < 0 || k >= len(arr) {
			return nil, fmt.Errorf("索引 %d 越界，数组长度 %d", k, len(arr))
		}
		return arr[k], nil
	case float64: // JSON 数字默认 float64，转为 int 处理
		return n.getCurrentValue(int(k))
	default:
		return nil, fmt.Errorf("不支持的键类型 %T，仅支持 string 或 int", key)
	}
}

// Result 返回最近一次 Get 的结果。
func (n *JSONNavigator) Result() any {
	return n.lastResult
}

// Gets 批量导航：根据路径，在每一层使用 GetCurs 进行批量提取。
// 返回 *JSONNavigator 支持链式调用，错误通过 Error() 获取。
func (n *JSONNavigator) Gets(path []any) *JSONNavigator {
	if n.err != nil {
		return n
	}
	if len(path) == 0 {
		n.setError(fmt.Errorf("路径不能为空"))
		return n
	}

	n.current = n.root

	for _, key := range path {
		results, err := n.getCursValues(key)
		if err != nil {
			n.setError(err)
			return n
		}
		n.current = results // 当前层变为结果数组
	}
	n.lastResults = n.current.([]any)
	return n
}

// getCursValues 从当前数组批量提取值（内部方法）。
// key 支持 string、int、float64（兼容），含义与原设计一致。
func (n *JSONNavigator) getCursValues(key any) ([]any, error) {
	arr, ok := n.current.([]any)
	if !ok {
		return nil, fmt.Errorf("当前不是数组，无法批量提取")
	}

	var result []any
	switch k := key.(type) {
	case string:
		for i, item := range arr {
			obj, ok := item.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("数组第 %d 个元素不是对象，无法获取字符串键", i)
			}
			val, exists := obj[k]
			if !exists {
				return nil, fmt.Errorf("数组第 %d 个元素缺少键 %q", i, k)
			}
			result = append(result, val)
		}
	case int:
		if k == -1 {
			for i, item := range arr {
				subArr, ok := item.([]any)
				if !ok {
					return nil, fmt.Errorf("数组第 %d 个元素不是数组，无法展开", i)
				}
				result = append(result, subArr...)
			}
		} else {
			for i, item := range arr {
				subArr, ok := item.([]any)
				if !ok {
					return nil, fmt.Errorf("数组第 %d 个元素不是数组，无法按索引提取", i)
				}
				if k < 0 || k >= len(subArr) {
					return nil, fmt.Errorf("数组第 %d 个元素的索引 %d 越界", i, k)
				}
				result = append(result, subArr[k])
			}
		}
	case float64:
		return n.getCursValues(int(k))
	default:
		return nil, fmt.Errorf("不支持的键类型 %T", key)
	}
	return result, nil
}

// Results 返回最近一次 Gets 的结果切片。
func (n *JSONNavigator) Results() []any {
	return n.lastResults
}

// Reset 重置导航器到初始状态，并清除错误。
func (n *JSONNavigator) Reset() {
	n.current = n.root
	n.lastResult = nil
	n.lastResults = nil
	n.err = nil
}

// String 返回当前值的 JSON 字符串表示（调试用）。
func (n *JSONNavigator) String() string {
	b, _ := json.Marshal(n.current)
	return string(b)
}