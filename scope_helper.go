package dig

import "reflect"

var _ callFn = defaultCallFn

func defaultCallFn(fn reflect.Value, args []reflect.Value) []reflect.Value {
	return fn.Call(args)
}
