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
	typ    reflect.Type
	fields []parameterObjectField
}

var _ func(reflect.Type) (parameterObject, error) = newParameterObject

// parameterObjectField is a single field of a parameter struct.
type parameterObjectField struct {
	// fieldName is the name of the field in the struct.
	fieldName string

	// fieldIndex is the index of this field in the target struct.
	//
	// We need to track this separately because not all fields of the
	// struct map to params.
	fieldIndex int

	// param is the dependency requested by this field.
	param param
}

var _ func(int, reflect.StructField) (parameterObjectField, error) = newParameterObjectField

func (po parameterObject) String() string {
	fields := make([]string, len(po.fields))
	for i, f := range po.fields {
		fields[i] = f.param.String()
	}
	return strings.Join(fields, " ")
}

func (po parameterObject) Requires() resourceRequirements {
	var rr resourceRequirements

	for _, f := range po.fields {
		requiredKeys := f.param.Requires().required
		optionalKeys := f.param.Requires().optional

		rr.required = append(rr.required, requiredKeys...)
		rr.optional = append(rr.optional, optionalKeys...)
	}

	return rr
}

func (po parameterObject) LookupValue(sp *Scope) (reflect.Value, error) {
	dest := reflect.New(po.typ).Elem()
	for _, f := range po.fields {
		v, err := f.param.LookupValue(sp)
		if err != nil {
			return reflect.Value{}, err
		}
		dest.Field(f.fieldIndex).Set(v)
	}
	return dest, nil
}

func (po parameterObject) DotParams() []*dot.Param {
	var params []*dot.Param
	for _, f := range po.fields {
		params = append(params, f.param.DotParams()...)
	}
	return params
}

// newParameterObject builds a parameterObject from the provided type.
// The type MUST be a parameter struct.
func newParameterObject(typ reflect.Type) (parameterObject, error) {
	po := parameterObject{typ: typ}

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
		po.fields = append(po.fields, rof)
	}
	return po, nil
}

func newParameterObjectField(idx int, sf reflect.StructField) (parameterObjectField, error) {
	var p param
	switch {
	case sf.PkgPath != "":
		return parameterObjectField{}, newErrInvalidInput(
			fmt.Sprintf("unexported fields not allowed in dig.Required, "+
				"did you mean to export %q (%v)?", sf.Name, sf.Type), nil)

	case isErrorType(sf.Type):
		return parameterObjectField{}, newErrInvalidInput("cannot handle error here, return it from"+
			" the constructor instead", nil)

	case sf.Tag.Get(_groupTag) != "":
		var err error
		p, err = newValueParamGroup(sf)
		if err != nil {
			return parameterObjectField{}, err
		}

	default:
		var err error
		p, err = newParam(sf.Type)
		if err != nil {
			return parameterObjectField{}, err
		}
	}

	if ps, ok := p.(valueParam); ok {
		ps.valueName = sf.Tag.Get(_nameTag)

		var err error
		ps.optional, err = isFieldOptional(sf)
		if err != nil {
			return parameterObjectField{}, err
		}

		p = ps
	}

	return parameterObjectField{
		fieldName:  sf.Name,
		fieldIndex: idx,
		param:      p,
	}, nil
}
