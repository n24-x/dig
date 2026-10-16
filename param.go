package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

type param interface {
	fmt.Stringer

	Requires() []resourceKey

	// TODO
	// Build(containerStore) (reflect.Value, error)

	DotParam() []*dot.Param
}

var (
	_ param = (*valueParam)(nil)
	_ param = (*requiredObject)(nil)
	_ param = (*valueParamGroup)(nil)
)

var (
	_ func(reflect.Type) (param, error) = newParam
)

// newParam builds a param from the given typ. If the typ is a dig.Required
// struct, a requiredObject will be returned.
func newParam(typ reflect.Type) (param, error) {
	switch {
	case IsProvided(typ) ||
		(typ.Kind() == reflect.Pointer && IsProvided(typ.Elem())) ||
		embedsType(typ, _providedPtrType):

		return nil, newErrInvalidInput(
			fmt.Sprintf(
				"cannot depend on result objects: %v embeds a dig.Provided",
				typ,
			), nil)

	case embedsType(typ, _requiredPtrType):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"cannot build a parameter object by embedding *dig.Required, embed dig.Required instead: %v embeds *dig.Required",
			typ), nil)

	case typ.Kind() == reflect.Pointer && IsRequired(typ.Elem()):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"cannot depend on a pointer to a parameter object, use a value instead: %v is a pointer to a struct that embeds dig.Required", typ), nil)

	case IsRequired(typ):
		return newRequiredObject(typ)

	default:
		return newValueParam(typ), nil
	}
}
