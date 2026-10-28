package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// The result interface represents a result produced by a constructor.
type result interface {
	// Provides reports the keys this result fills as a provider.
	Provides() []resourceType

	// DecorationKeys reports the keys this result takes over as a decorator.
	//
	// It differs from Provides only for a group. A provider may add one member
	// to a group, but a decorator replaces the group, and readers look a group
	// up by its element type, so a decoration has to land there. A result that
	// cannot stand for a whole group is rejected here.
	DecorationKeys() ([]resourceType, error)

	StoreValue(w valueWriter, decorated bool, v reflect.Value)

	DotResults() []*dot.Result
}

var (
	_ result = valueResult{}
	_ result = returnObject{}
	_ result = valueResultGroup{}
)

var _ func(reflect.Type, resultOptions) (result, error) = newResult

// newResult builds a result from the given typ.
func newResult(typ reflect.Type, opts resultOptions) (result, error) {
	switch {
	case IsRequired(typ) ||
		(typ.Kind() == reflect.Pointer && IsRequired(typ.Elem())) ||
		embedsType(typ, _requiredPtrType):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"expected return-struct, but got parameter-struct: %v embeds a dig.Required", typ), nil)

	case isErrorType(typ):
		return nil, newErrInvalidInput("cannot handle error here, return it from the constructor instead", nil)

	case IsProvided(typ):
		return newReturnObject(typ, opts)

	case embedsType(typ, _providedPtrType):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"return-struct cannot embed *dig.Provided, "+
				"embed dig.Provided instead: %v embeds *dig.Provided", typ), nil)

	case typ.Kind() == reflect.Pointer && IsProvided(typ.Elem()):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"cannot use pointer as return-struct, use a value instead:"+
				" %v is a pointer to a struct that embeds dig.Provided", typ), nil)

	case opts.groupTag != "":
		g, err := parseGroupString(opts.groupTag)
		if err != nil {
			return nil, newErrInvalidInput(
				fmt.Sprintf("cannot parse group-tag value %q", opts.groupTag), err)
		}
		rg := valueResultGroup{valueType: typ, groupName: g.name, flatten: g.flatten}

		// A group holds one value per member and is read as the whole slice.
		// As changes what a single result is keyed by, which for a group would
		// mean keying the member by another type: either the group stops being
		// the one consumers read, or flatten has already turned the member into
		// the element and there is nothing left to cast. Neither is useful, so
		// reject the combination instead of silently keying the group away.
		if len(opts.asTypes) > 0 {
			return nil, newErrInvalidInput(fmt.Sprintf(
				"cannot use dig.As with a group: %v is added to group %q as a whole, "+
					"so there is no single type for As to apply to", typ, g.name), nil)
		}

		if g.passive {
			return nil, newErrInvalidInput(fmt.Sprintf(
				"cannot use group-tag with passive in return-struct:"+
					" passive was incorrectly used in group %q", g.name), nil)
		}
		if g.flatten {
			if typ.Kind() != reflect.Slice {
				return nil, newErrInvalidInput(fmt.Sprintf(
					"in return-struct, only slice field can use group-tag with flatten:"+
						" %v is not a slice", typ), nil)
			}
			rg.valueType = rg.valueType.Elem()
		}
		return rg, nil
	default:
		return newValueResult(typ, opts)
	}
}
