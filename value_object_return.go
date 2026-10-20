package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

type returnObject struct {
	Type   reflect.Type
	Fields []returnObjectField
}

var _ func(reflect.Type, resultOptions) (returnObject, error) = newReturnObject

type returnObjectField struct {
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

var _ func(int, reflect.StructField, resultOptions) (returnObjectField, error) = newReturnObjectField

func (po returnObject) Provides() []resourceType {
	if len(po.Fields) == 0 {
		return []resourceType{}
	}

	var keys []resourceType
	for _, f := range po.Fields {
		keys = append(keys, f.Result.Provides()...)
	}

	return keys
}

func (po returnObject) DotResults() []*dot.Result {
	var dotResults []*dot.Result
	for _, f := range po.Fields {
		dotResults = append(dotResults, f.Result.DotResults()...)
	}
	return dotResults
}

func newReturnObject(typ reflect.Type, opts resultOptions) (returnObject, error) {
	po := returnObject{
		Type: typ,
	}

	if opts.ValueName != "" {
		return returnObject{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use dig.Name with return-struct: %v embeds dig.Provided", typ), nil)
	}
	if opts.GroupName != "" {
		return returnObject{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use dig.Group with return-struct: %v embeds dig.Provided", typ), nil)
	}
	if len(opts.As) > 0 {
		return returnObject{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use dig.As with return-struct: %v embeds dig.Provided", typ), nil)
	}

	for i := range typ.NumField() {
		f := typ.Field(i)
		if f.Type == _providedType {
			continue // skip dig.Provided
		}

		pof, err := newReturnObjectField(i, f, opts)
		if err != nil {
			return returnObject{}, newErrInvalidInput(
				fmt.Sprintf("bad field %q of %v", f.Name, typ), err)
		}

		po.Fields = append(po.Fields, pof)
	}
	return po, nil
}

func newReturnObjectField(index int, sf reflect.StructField, opts resultOptions) (
	returnObjectField,
	error,
) {
	var r result
	switch {
	case isError(sf.Type):
		return returnObjectField{}, newErrInvalidInput("cannot handle error here, return it from"+
			" the constructor instead", nil)
	case sf.PkgPath != "":
		return returnObjectField{}, newErrInvalidInput(
			fmt.Sprintf("unexported fields not allowed in return-struct, did you mean to export %q (%v)?",
				sf.Name, sf.Type), nil)

	case sf.Tag.Get(_optionalTag) != "":
		return returnObjectField{}, newErrInvalidInput(
			fmt.Sprintf("cannot use optional-tag in return-struct: "+
				"field %q (%v) has optional-tag",
				sf.Name,
				sf.Type,
			), nil)

	case sf.Tag.Get(_groupTag) != "":
		var err error
		r, err = newValueResultGroup(sf)
		if err != nil {
			return returnObjectField{}, err
		}

	default:
		var err error
		if name := sf.Tag.Get(_nameTag); name != "" {
			// can modify in-place because options are passed-by-value.
			opts.ValueName = name
		}
		r, err = newResult(sf.Type, opts)
		if err != nil {
			return returnObjectField{}, err
		}
	}

	return returnObjectField{
		FieldName:  sf.Name,
		FieldIndex: index,
		Result:     r,
	}, nil
}
