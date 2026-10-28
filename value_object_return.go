package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

type returnObject struct {
	fields []returnObjectField
}

var _ func(reflect.Type, resultOptions) (returnObject, error) = newReturnObject

type returnObjectField struct {
	// fieldName is the name of the field in the struct.
	fieldName string

	// fieldIndex is the index of the field in the struct.
	//
	// We need to track this separately because not all fields of the struct
	// map to results.
	fieldIndex int

	// result is what this field produces.
	result result
}

var _ func(int, reflect.StructField, resultOptions) (returnObjectField, error) = newReturnObjectField

func (ro returnObject) Provides() []resourceType {
	if len(ro.fields) == 0 {
		return []resourceType{}
	}

	var keys []resourceType
	for _, f := range ro.fields {
		keys = append(keys, f.result.Provides()...)
	}

	return keys
}

func (ro returnObject) StoreValue(w valueWriter, decorated bool, v reflect.Value) {
	for _, f := range ro.fields {
		f.result.StoreValue(w, decorated, v.Field(f.fieldIndex))
	}
}

func (ro returnObject) DotResults() []*dot.Result {
	var dotResults []*dot.Result
	for _, f := range ro.fields {
		dotResults = append(dotResults, f.result.DotResults()...)
	}
	return dotResults
}

func newReturnObject(typ reflect.Type, opts resultOptions) (returnObject, error) {
	var ro returnObject

	if opts.nameTag != "" {
		return returnObject{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use dig.Name with return-struct: %v embeds dig.Provided", typ), nil)
	}
	if opts.groupTag != "" {
		return returnObject{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use dig.Group with return-struct: %v embeds dig.Provided", typ), nil)
	}

	for i := range typ.NumField() {
		f := typ.Field(i)
		if f.Type == _providedType {
			continue // skip dig.Provided
		}

		rof, err := newReturnObjectField(i, f, opts)
		if err != nil {
			return returnObject{}, newErrInvalidInput(
				fmt.Sprintf("bad field %q of %v", f.Name, typ), err)
		}

		ro.fields = append(ro.fields, rof)
	}
	return ro, nil
}

func newReturnObjectField(index int, sf reflect.StructField, opts resultOptions) (
	returnObjectField,
	error,
) {
	var r result
	switch {
	case isErrorType(sf.Type):
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
		rg, err := newValueResultGroup(sf)
		if err != nil {
			return returnObjectField{}, err
		}

		// A group is added as a whole, so As has no single type to apply to.
		// Checked after the group itself is parsed so that a bad tag reports
		// its own error instead of this one.
		if len(opts.asTypes) > 0 {
			return returnObjectField{}, newErrInvalidInput(fmt.Sprintf(
				"cannot use dig.As with a group field: %v is added to group %q as a whole, "+
					"so there is no single type for As to apply to", sf.Type, rg.groupName), nil)
		}
		r = rg

	default:
		var err error
		if name := sf.Tag.Get(_nameTag); name != "" {
			// can modify in-place because options are passed-by-value.
			opts.nameTag = name
		}
		r, err = newResult(sf.Type, opts)
		if err != nil {
			return returnObjectField{}, err
		}
	}

	return returnObjectField{
		fieldName:  sf.Name,
		fieldIndex: index,
		result:     r,
	}, nil
}
