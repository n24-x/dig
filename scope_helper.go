package dig

import "reflect"

var _ callFn = defaultCallFn
var _ callFn = dryCallFn

func defaultCallFn(fn reflect.Value, args []reflect.Value) []reflect.Value {
	return fn.Call(args)
}

// Generates zero values for results without calling the given function.
func dryCallFn(fn reflect.Value, _ []reflect.Value) []reflect.Value {
	ftype := fn.Type()
	results := make([]reflect.Value, ftype.NumOut())
	for i := 0; i < ftype.NumOut(); i++ {
		results[i] = reflect.Zero(fn.Type().Out(i))
	}

	return results
}
