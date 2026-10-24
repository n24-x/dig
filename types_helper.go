package dig

import (
	"reflect"
)

var (
	_noValue         reflect.Value
	_errType         = reflect.TypeFor[error]()
	_requiredType    = reflect.TypeFor[Required]()
	_requiredPtrType = reflect.TypeFor[*Required]()
	_providedType    = reflect.TypeFor[Provided]()
	_providedPtrType = reflect.TypeFor[*Provided]()
)

func isErrorType(t reflect.Type) bool {
	return t != nil && t.Implements(_errType)
}

func isErrorValue(v reflect.Value) bool {
	err, _ := reflect.TypeAssert[error](v)
	if err != nil {
		return true
	}
	return false
}

func findError(out []reflect.Value) error {
	for _, o := range out {
		if err, ok := reflect.TypeAssert[error](o); ok {
			return err
		}
	}
	return nil
}

// embedsType reports whether the type of value embeds target,
// or whether any type embedded by it embeds target.
func embedsType(value any, target reflect.Type) bool {
	// TODO: this function doesn't consider e being a pointer.
	// given `type A foo { *Required }`, this function would return false for
	// embedding dig.Required, which makes for some extra error checking in places
	// that call this function. Might be worthwhile to consider reflect.Indirect
	// usage to clean up the callers.

	if value == nil {
		return false
	}

	// value may already be a reflect.Type; otherwise derive its type.
	root, ok := value.(reflect.Type)
	if !ok {
		root = reflect.TypeOf(value)
	}

	// Walk the embedded-type graph (DFS).
	stack := []reflect.Type{root}
	for len(stack) > 0 {
		typ := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if typ == target {
			return true
		}
		if typ.Kind() != reflect.Struct {
			continue
		}
		for f := range typ.Fields() {
			if f.Anonymous {
				stack = append(stack, f.Type)
			}
		}
	}

	return false
}

var (
	_ func(reflect.Type) bool      = isErrorType
	_ func(any, reflect.Type) bool = embedsType
)
