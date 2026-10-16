package dig

import "reflect"

// valueResult is an explicit value produced by a constructor, optionally
// with a name and/or extra As types.
type valueResult struct {
	ValueName string
	ValueType reflect.Type

	As []reflect.Type
}

func (r valueResult) Provides() []resourceKey {
	keys := []resourceKey{
		{
			valueType: r.ValueType,
			valueName: r.ValueName,
		},
	}
	for _, typ := range r.As {
		keys = append(keys, resourceKey{
			valueType: typ,
			valueName: r.ValueName,
		})
	}
	return keys
}
