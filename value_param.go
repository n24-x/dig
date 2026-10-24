package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// param is one argument of a function, resolved against a scope.
type param interface {
	fmt.Stringer

	// Requires reports the resources this param depends on, split by whether
	// the graph has to guarantee them.
	Requires() resourceRequirements

	// Value produces the value this param stands for, reading whatever the
	// scope already holds.
	//
	// It does not execute constructors. By the time Value runs, everything a
	// required param needs has been built, so a required param that finds
	// nothing has hit an invariant violation (errValueNotFound) rather than a
	// missing dependency. Optional params fall back to a zero value, groups
	// assemble the items collected for them, and parameter structs build
	// themselves field by field.
	Value(*Scope) (reflect.Value, error)

	DotParams() []*dot.Param
}

type resourceRequirements struct {
	Required []resourceType
	Optional []resourceType
	// TODO Passive []resource
}

var (
	_ param = valueParam{}
	_ param = parameterObject{}
	_ param = valueParamGroup{}
)

var _ func(reflect.Type) (param, error) = newParam

// newParam builds a param from the given typ. If the typ is a parameter
// struct, a parameterObject will be returned.
func newParam(typ reflect.Type) (param, error) {
	switch {
	case IsProvided(typ) ||
		(typ.Kind() == reflect.Pointer && IsProvided(typ.Elem())) ||
		embedsType(typ, _providedPtrType):

		return nil, newErrInvalidInput(
			fmt.Sprintf(
				"expected parameter-struct, but got return-struct: %v embeds a dig.Provided",
				typ,
			), nil)
	case isErrorType(typ):
		return nil, newErrInvalidInput("cannot handle error here, return it from the constructor instead", nil)

	case embedsType(typ, _requiredPtrType):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"parameter-struct cannot embed *dig.Required, embed dig.Required instead: %v embeds *dig.Required",
			typ), nil)

	case typ.Kind() == reflect.Pointer && IsRequired(typ.Elem()):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"cannot use a pointer as parameter-struct, use a value instead:"+
				" %v is a pointer to a struct that embeds dig.Required", typ), nil)

	case IsRequired(typ):
		return newParameterObject(typ)

	default:
		return newValueParam(typ), nil
	}
}
