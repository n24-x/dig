package dig

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/n24-x/dig/internal/dot"
)

// parameterObject is a parameter struct where each field is another param.
//
// This object is not expected in the graph as-is.
type parameterObject struct {
	Type        reflect.Type
	Fields      []parameterObjectField
	FieldOrders []int
}

var _ func(reflect.Type) (parameterObject, error) = newParameterObject

// parameterObjectField is a single field of a parameter struct.
type parameterObjectField struct {
	// Name of the field in the struct.
	FieldName string

	// Index of this field in the target struct.
	//
	// We need to track this separately because not all fields of the
	// struct map to params.
	FieldIndex int

	// The dependency requested by this field.
	Param param
}

var _ func(int, reflect.StructField) (parameterObjectField, error) = newParameterObjectField

func (ro parameterObject) String() string {
	fields := make([]string, len(ro.Fields))
	for i, f := range ro.Fields {
		fields[i] = f.Param.String()
	}
	return strings.Join(fields, " ")
}

func (ro parameterObject) Requires() []resourceID {
	var keys []resourceID
	for _, f := range ro.Fields {
		keys = append(keys, f.Param.Requires()...)
	}
	return keys
}

func (ro parameterObject) DotParams() []*dot.Param {
	var params []*dot.Param
	for _, f := range ro.Fields {
		params = append(params, f.Param.DotParams()...)
	}
	return params
}

// newParameterObject builds a parameterObject from the provided type.
// The type MUST be a parameter struct.
func newParameterObject(typ reflect.Type) (parameterObject, error) {
	ro := parameterObject{Type: typ}

	var ignoreUnexported bool
	for f := range typ.Fields() {
		if f.Type == _requiredType {
			var err error
			ignoreUnexported, err = isIgnoreUnexportedSet(f)
			if err != nil {
				return parameterObject{}, err
			}
			break
		}
	}

	for i := range typ.NumField() {
		f := typ.Field(i)
		if f.Type == _requiredType {
			// Skip dig.Required.
			continue
		}
		if f.PkgPath != "" && ignoreUnexported {
			// Skip over an unexported field if it is allowed.
			continue
		}
		rof, err := newParameterObjectField(i, f)
		if err != nil {
			return parameterObject{}, newErrInvalidInput(
				fmt.Sprintf("bad field %q of %v", f.Name, typ), err)
		}
		ro.Fields = append(ro.Fields, rof)
	}
	return ro, nil
}

func newParameterObjectField(idx int, f reflect.StructField) (parameterObjectField, error) {
	pof := parameterObjectField{
		FieldName:  f.Name,
		FieldIndex: idx,
	}

	var p param
	switch {
	case f.PkgPath != "":
		return pof, newErrInvalidInput(
			fmt.Sprintf("unexported fields not allowed in dig.Required, "+
				"did you mean to export %q (%v)?", f.Name, f.Type), nil)

	case f.Tag.Get(_groupTag) != "":
		var err error
		p, err = newValueParamGroup(f)
		if err != nil {
			return pof, err
		}

	default:
		var err error
		p, err = newParam(f.Type)
		if err != nil {
			return pof, err
		}
	}

	if ps, ok := p.(valueParam); ok {
		ps.ValueName = f.Tag.Get(_nameTag)

		var err error
		ps.Optional, err = isFieldOptional(f)
		if err != nil {
			return parameterObjectField{}, err
		}

		p = ps
	}

	pof.Param = p
	return pof, nil
}
