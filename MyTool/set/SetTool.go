package set

import (
	"encoding/json"
	"fmt"
)

type Set struct {
	data map[string]any
}
type setKey struct {
	Key any
}

func NewSet() *Set {
	p := new(Set)
	p.data = make(map[string]any)
	return p
}
func ArrayToSet(arr []any) *Set {
	return NewSet().AddArr(arr)
}

func anyToJsonString(v any) string {
	//cmd.Log("anyToJsonString v", v)
	vv := setKey{Key: v}
	b, e := json.Marshal(vv)
	if e != nil {
		//cmd.Log("anyToJsonString error", e)
		panic(e)
		return ""
	}
	//cmd.Log("anyToJsonString b", string(b))
	return string(b)
}
func JsonStringToAny(js string) any {
	v := setKey{}
	json.Unmarshal([]byte(js), &v)
	return v.Key
}
func (this *Set) ToAnyArray() []any {
	arr := make([]any, 0)
	for k, _ := range this.data {
		arr = append(arr, JsonStringToAny(k))
	}
	return arr
}
func (this *Set) Add(v any) *Set {
	this.data[anyToJsonString(v)] = nil
	return this
}
func (this *Set) AddArr(arr []any) *Set {
	for _, v := range arr {
		this.Add(v)
	}
	return this
}
func (this *Set) AddSet(arr Set) *Set {
	for k, _ := range arr.data {
		this.data[k] = nil
	}
	return this
}
func (this *Set) RemoveArr(arr []any) *Set {
	for _, v := range arr {
		delete(this.data, anyToJsonString(v))
	}
	return this
}
func (this *Set) RemoveSet(arr Set) *Set {
	for k, _ := range arr.data {
		delete(this.data, k)
	}
	return this
}

func (this *Set) Remove(v any) *Set {
	delete(this.data, anyToJsonString(v))
	return this
}
func (this *Set) Contain(v any) bool {
	_, ok := this.data[anyToJsonString(v)]
	return ok
}
func (this *Set) Size() int {
	return len(this.data)
}
func (this *Set) CopySelf() *Set { //返回新的set
	newSet := NewSet()
	for k, _ := range this.data {
		newSet.data[k] = nil
	}
	return newSet
}
func (this *Set) Intersection(other Set) *Set { //返回新的交集set
	newSet := NewSet()
	for k, _ := range newSet.data {
		if other.Contain(k) {
			newSet.data[k] = nil
		}
	}
	return newSet
}
func (this *Set) Merge(other Set) *Set { //返回新的并集set
	newSet := this.CopySelf()
	newSet.AddSet(other)
	return newSet
}
func (this *Set) Difference(other Set) *Set { //返回新的差集set
	newSet := this.CopySelf()
	newSet.RemoveSet(other)
	return newSet
}
func (this *Set) Print() *Set {
	fmt.Printf(">>>set %#v\n", this.data)
	return this
}
