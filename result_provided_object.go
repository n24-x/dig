package dig

import "reflect"

type providedObject struct {
	Type   reflect.Type
	Fields []providedObjectField
}

type providedObjectField struct {
	// Name of the field in the struct.
	FieldName string

	// Index of the field in the struct.
	//
	// We need to track this separately because not all fields of the struct
	// map to results.
	FieldIndex int

	// Result produced by this field.
	Result result
}

func (po providedObject) Provides() []resourceKey {
	if len(po.Fields) == 0 {
		return []resourceKey{}
	}

	var keys []resourceKey
	for _, f := range po.Fields {
		keys = append(keys, f.Result.Provides()...)
	}

	return keys
}
