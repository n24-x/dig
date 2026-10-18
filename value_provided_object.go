package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

type providedObject struct {
	Type   reflect.Type
	Fields []providedObjectField
}

var _ func(reflect.Type, resultOptions) (providedObject, error) = newProvidedObject

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

var _ func(int, reflect.StructField, resultOptions) (providedObjectField, error) = newProvidedObjectField

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

func (po providedObject) DotResults() []*dot.Result {
	var dotResults []*dot.Result
	for _, f := range po.Fields {
		dotResults = append(dotResults, f.Result.DotResults()...)
	}
	return dotResults
}

func newProvidedObject(typ reflect.Type, opts resultOptions) (providedObject, error) {
	po := providedObject{
		Type: typ,
	}

	if opts.Name != "" {
		return providedObject{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use dig.Name with return-struct: %v embeds dig.Provided", typ), nil)
	}
	if opts.Group != "" {
		return providedObject{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use dig.Group with return-struct: %v embeds dig.Provided", typ), nil)
	}
	if len(opts.As) > 0 {
		return providedObject{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use dig.As with return-struct: %v embeds dig.Provided", typ), nil)
	}

	for i := range typ.NumField() {
		f := typ.Field(i)
		if f.Type == _providedType {
			continue // skip dig.Provided
		}

		pof, err := newProvidedObjectField(i, f, opts)
		if err != nil {
			return providedObject{}, newErrInvalidInput(
				fmt.Sprintf("bad field %q of %v", f.Name, typ), err)
		}

		po.Fields = append(po.Fields, pof)
	}
	return po, nil
}

func newProvidedObjectField(index int, sf reflect.StructField, opts resultOptions) (
	providedObjectField,
	error,
) {
	pof := providedObjectField{
		FieldName:  sf.Name,
		FieldIndex: index,
	}

	var r result
	switch {
	case sf.PkgPath != "":
		return providedObjectField{}, newErrInvalidInput(
			fmt.Sprintf("unexported fields not allowed in return-struct, did you mean to export %q (%v)?",
				sf.Name, sf.Type), nil)

	case sf.Tag.Get(_optionalTag) != "":
		return providedObjectField{}, newErrInvalidInput(
			fmt.Sprintf("cannot use optional-tag in return-struct: "+
				"field %q (%v) has optional-tag",
				sf.Name,
				sf.Type,
			), nil)

	case sf.Tag.Get(_groupTag) != "":
		var err error
		r, err = newValueResultGroup(sf)
		if err != nil {
			return providedObjectField{}, err
		}

	default:
		var err error
		if name := sf.Tag.Get(_nameTag); name != "" {
			// can modify in-place because options are passed-by-value.
			opts.Name = name
		}
		r, err = newResult(sf.Type, opts)
		if err != nil {
			return providedObjectField{}, err
		}
	}

	pof.Result = r
	return pof, nil
}
