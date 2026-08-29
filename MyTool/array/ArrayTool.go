package array

func FindIndex[T comparable](el T, arr []T) []int { //返回index
	var index []int
	for i, v := range arr {
		if v == el {
			index = append(index, i)
		}
	}
	return index
}
func Include[T comparable](arr1 []T, arr2 []T) bool {
	for _, v := range arr1 {
		if len(FindIndex(v, arr2)) != 0 {
			return true
		}
	}
	return false
}

func AnyToType[T comparable](arr []any) []T {
	var res []T
	for _, v := range arr {
		res = append(res, v.(T))
	}
	return res
}
func TypeToAny[T comparable](arr []T) []any {
	var res []any
	for _, v := range arr {
		res = append(res, v)
	}
	return res
}
