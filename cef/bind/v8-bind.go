//----------------------------------------
//
// Copyright © yanghy. All Rights Reserved.
//
// Licensed under Apache License Version 2.0, January 2004
//
// https://www.apache.org/licenses/LICENSE-2.0
//
//----------------------------------------

package bind

import (
	"container/list"
	"fmt"
	"github.com/energye/energy/pkgs/json"
	"sync"
	"unsafe"
)

var bind = &v8bind{hasFieldCollection: make(map[string]uintptr), fieldCollection: list.New()}

type v8bind struct {
	hasFieldCollection map[string]uintptr
	fieldCollection    *list.List
	lock               sync.Mutex
}

// Set 添加或修改
//
//	参数
//		 name: 唯一字段名, 重复将被覆盖
//		value: 值
func (m *v8bind) Set(name string, value JSValue) {
	if id, ok := m.hasFieldCollection[name]; ok {
		old := m.Remove(id)
		id = m.Add(value)
		value.setId(id)
		m.hasFieldCollection[name] = id
		switch old.(type) {
		case JSValue:
			old.(JSValue).setId(id)
		}
	} else {
		id = m.Add(value)
		value.setId(id)
		m.hasFieldCollection[name] = id
	}
	//m.fieldCollection[name] = value
}

func (m *v8bind) GetJSValue(id uintptr) JSValue {
	if v := m.Get(id); v != nil {
		return v.Value.(JSValue)
	}
	return nil
}

func (m *v8bind) Add(value JSValue) uintptr {
	return uintptr(unsafe.Pointer(m.fieldCollection.PushBack(value)))
}

func (m *v8bind) Get(id uintptr) *list.Element {
	return (*list.Element)(unsafe.Pointer(id))
}

func (m *v8bind) Remove(id uintptr) any {
	if v := m.Get(id); v != nil {
		r := m.fieldCollection.Remove(v)
		v.Value = nil
		return r
	}
	return nil
}

func (m *v8bind) Binds() map[string]JSValue {
	//return m.fieldCollection
	return nil
}

func GetBinds(fn func(binds map[string]JSValue)) {
	//fn(bind.fieldCollection)
}

func Test() {
	//字段
	stringKey0 := NewString("stringKey", "字符串值0")
	fmt.Println("stringKey", stringKey0, stringKey0.Value())
	stringKey1 := NewString("stringKey", "字符串值1")
	integerKey := NewInteger("integerKey", 1000)
	fmt.Println("stringKey", stringKey0)
	fmt.Println("stringKey", stringKey1, stringKey1.Value())
	fmt.Println("integerKey", integerKey.Value())
	integerKey.SetValue("变成字符串")
	fmt.Println("integerKey", integerKey.AsString().Value())
	integerKey.SetValue(true)
	fmt.Println("integerKey", integerKey.AsBoolean().Value())
	boolField := integerKey.AsBoolean()
	fmt.Println("boolField", boolField.Value())
	fmt.Println("boolField", bind.GetJSValue(boolField.Id()).AsBoolean().Value())
	boolField.SetValue(false)
	fmt.Println("boolField", bind.GetJSValue(boolField.Id()).AsBoolean().Value())
	fmt.Println(bind.fieldCollection.Len())

	//函数
	funcKey := NewFunction("funcKey", func(in1 string) {
		fmt.Println(in1)
	})
	inArgument := json.NewJSONArray(nil)
	inArgument.Add("字符串参数")
	funcKey.Invoke(inArgument)

	funcKey.SetValue("函数变字符串")
	funcToString := funcKey.AsString()
	fmt.Println("funcToString:", funcToString.Value())

	// 对象
	type objectDemo1 struct {
		Key1 string
		Key2 string
	}
	type objectDemo2 struct {
		Key1 string
		Key2 string
		Key3 int
	}
	type object struct {
		Key1 string
		Key2 string
		Key3 int
		Key4 float64
		Key5 bool
		Key6 *objectDemo1
		Key7 objectDemo2
		Key8 []string
	}
	var testObj = &object{
		Key1: "value1",
		Key2: "value2",
		Key3: 333,
		Key4: 555.3,
		Key5: true,
		//Key6: &objectDemo1{},
	}

	objectKey := NewObject(testObj)
	fmt.Println("objectKey:", objectKey.JSONString())
	objectKey.Set("Key1", "值1")
	objectKey.Set("Key2", "值2")
	objectKey.Set("Key3", 4444)
	objectKey.Set("Key4", 9999.99)
	objectKey.Set("Key5", false)
	objectKey.Set("Key6", &objectDemo1{Key1: "Key6值"})
	objectKey.Set("Key7", objectDemo2{Key1: "值值"})
	fmt.Println("objectKey:", objectKey.JSONString())
	objectKey1 := objectKey.Get("Key1")
	fmt.Println("objectKey1Name:", objectKey1.Name())
	objectKey1.SetValue("objectKey1设置新值 ")
	fmt.Println("objectKey1:", objectKey1.JSONString())
	objectKey6 := objectKey.Get("Key6")
	fmt.Println("objectKey:", objectKey.JSONString())
	objectKey6.SetValue(&objectDemo1{Key1: "objectKey6Key1"})
	fmt.Println("objectKey6:", objectKey6.JSONString())
	fmt.Println("objectKey:", objectKey.JSONString())
	objectKey.Set("Key8", []string{"v1", "v2"})
	objectKey8 := objectKey.Get("Key8")
	fmt.Println("objectKey8:", objectKey8.JSONString())
	objectKey8.AsArray()
	//object end
	fmt.Println("objectKey:", objectKey.JSONString())
	//object to string
	objectKey.SetValue("对象变成字符串")
	objectToString := objectKey.AsString()
	fmt.Println("objectToString:", objectToString.Value())
	fmt.Println("objectKey:", objectKey.JSONString())
	objectKey.SetValue(testObj)
	fmt.Println("objectKey:", objectKey.JSONString())

	// 数组
}

func TestBind() {

}
