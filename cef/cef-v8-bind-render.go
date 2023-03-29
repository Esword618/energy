//----------------------------------------
//
// Copyright © yanghy. All Rights Reserved.
//
// Licensed under Apache License Version 2.0, January 2004
//
// https://www.apache.org/licenses/LICENSE-2.0
//
//----------------------------------------

// 基于IPC的字段数据绑定 - 渲染(子)进程
package cef

import (
	"bytes"
	"fmt"
	"github.com/energye/energy/cef/bind"
	"github.com/energye/energy/pkgs/json"
	"strings"
	"text/template"
)

var bindRender *bindRenderProcess

type bindRenderProcess struct {
	isInitBindIPC bool
	handler       *ICefV8Handler
}

func (m *bindRenderProcess) initBindIPC() {
	if m.isInitBindIPC {
		return
	}
	m.isInitBindIPC = true
	renderIPC.addCallback(func(channelId int64, data json.JSON) bool {
		if data != nil {
			//messageJSON := data.JSONObject()
			//messageId := messageJSON.GetIntByKey(ipc_id)// messageId: 同步永远是1
			//name := messageJSON.GetStringByKey(ipc_name)
			//argumentList := messageJSON.GetArrayByKey(ipc_argumentList)
			//if name == internalIPCJSExecuteGoSyncEventReplay {
			//	return true
			//}
		}
		return false
	})
	renderIPC.ipc.Send(nil)
}

func (m *bindRenderProcess) webKitMakeBind() {
	//var nameBuild = &strings.Builder{}
	//var id = "p"
	//var typ = "t"
	//var objectFieldBind = func(bind *bind.V8bind, object json.JSONObject, jsv bind.JSValue) {
	//	names := strings.Split(jsv.Name(), ".")
	//	fmt.Println("object ElementToJSValue Object:", jsv.Name(), jsv.Type())
	//	//fmt.Println("object:", object.ToJSONString())
	//	if len(names) == 1 {
	//		if jsv.IsObject() {
	//			field := json.NewJSONObject(nil)
	//			field.Set(id, jsv.Id())
	//			field.Set(typ, jsv.Type())
	//			object.Set(jsv.Name(), field)
	//		} else if jsv.IsArray() {
	//			object.Set(jsv.Name(), json.NewJSONArray(nil))
	//		} else {
	//			field := json.NewJSONObject(nil)
	//			field.Set(id, jsv.Id())
	//			field.Set(typ, jsv.Type())
	//			object.Set(jsv.Name(), field)
	//		}
	//	} else {
	//		name := names[len(names)-1]
	//		var pObject = object
	//		for i := 0; i < len(names)-1; i++ {
	//			pObject = pObject.GetByKey(names[i])
	//		}
	//		field := json.NewJSONObject(nil)
	//		field.Set(id, jsv.Id())
	//		field.Set(typ, jsv.Type())
	//		if jsv.IsObject() {
	//			if pObject.IsArray() {
	//				pObject.JSONArray().Add(field)
	//			} else {
	//				pObject.Set(name, field)
	//			}
	//		} else if jsv.IsArray() {
	//			if pObject.IsObject() {
	//				pObject.Set(name, field)
	//			} else {
	//				pObject.JSONArray().Add(field)
	//			}
	//		} else if jsv.IsFunction() {
	//			if pObject.IsObject() {
	//				pObject.Set(name, field)
	//			} else {
	//				pObject.JSONArray().Add(field)
	//			}
	//		} else {
	//			if pObject.IsObject() {
	//				pObject.Set(name, field)
	//			} else if pObject.IsArray() {
	//				pObject.JSONArray().Add(field)
	//			}
	//		}
	//	}
	//}
	bind.GetBinds(func(bind *bind.V8bind) {
		var object = json.NewJSONObject(nil)
		//fields := bind.FieldCollection()
		//for item := fields.Front(); item != nil; item = item.Next() {
		//	jsv := bind.ElementToJSValue(item)
		//	objectFieldBind(bind, object, jsv)
		//}
		fmt.Println("object:", object.ToJSONString())
		fmt.Println("FieldCollection:", bind.FieldCollection())
	})
	m.handler = V8HandlerRef.New()
	m.handler.Execute(func(name string, object *ICefV8Value, arguments *TCefV8ValueArray, retVal *ResultV8Value, exception *ResultString) bool {
		fmt.Println("v8Handler.Execute", name, renderIPC, arguments.Size(), arguments.Get(0).GetIntValue())
		if renderIPC != nil {
			//renderIPC.ipc.Send()
		}
		return false
	})
	//注册js
	//	var jsCode = `
	//let bind;
	//if (!bind) {
	//	bind = {};
	//}
	//(function () {
	//	Object.defineProperty(bind, 'myparam', {
	//		get(){
	//			native function getMyParam();
	//			return getMyParam();
	//		},
	//		set(v){
	//			native function setMyParam();
	//			setMyParam(v);
	//		}
	//	});
	//})();
	//`
	//	fmt.Println("jsCode", jsCode)
	var buf = &bytes.Buffer{}
	buf.WriteString(fmt.Sprintf(`let %s=null;if(%s===null){%s={};}`, internalBind, internalBind, internalBind))
	var tempText = `
{{range $i, $v := $.fields}}
Object.defineProperty({{$.bind}}, "{{$v}}", {
	get(){
		native function {{getter ($v)}}(v);
		return {{getter ($v) }}(33333);
	},
	set(v){
		native function {{setter ($v)}}();
		{{setter ($v) }}(33333,v);
	}
});
{{end}}`
	var funcs = make(template.FuncMap)
	funcs["getter"] = func(v string) string {
		return "get" + strings.ToUpper(string(v[0])) + v[1:]
	}
	funcs["setter"] = func(v string) string {
		return "set" + strings.ToUpper(string(v[0])) + v[1:]
	}
	temp, err := template.New("v8bind").Funcs(funcs).Parse(tempText)
	if err != nil {
		panic(err)
	}
	var data = map[string]any{}
	data["bind"] = internalBind
	var field = []string{"myparam", "myparam1"}
	data["fields"] = field
	var tempResult = &bytes.Buffer{}
	err = temp.Execute(tempResult, data)
	if err != nil {
		panic(err)
	}
	buf.WriteString("(function(){")
	buf.Write(tempResult.Bytes())
	buf.WriteString("})();")
	fmt.Println("tempResult:", buf.String())
	registerExtension(fmt.Sprintf("%s/%s", internalV8Bind, internalBind), buf.String(), m.handler)
}
