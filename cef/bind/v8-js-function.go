//----------------------------------------
//
// Copyright © yanghy. All Rights Reserved.
//
// Licensed under Apache License Version 2.0, January 2004
//
// https://www.apache.org/licenses/LICENSE-2.0
//
//----------------------------------------

// V8 JSValue JSFunction 类型实现
package bind

type JSFunction interface {
	JSValue
	AsFunction() JSFunction
	Value() JSFunction
}

type jsFunction struct {
	V8Value
}

func (m *jsFunction) AsFunction() JSFunction {
	return m
}

func (m *jsFunction) Value() JSFunction {
	return nil
}
