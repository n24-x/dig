package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// The result interface represents a result produced by a constructor.
type result interface {
	Provides() []resourceKey

	// TODO
	// Extract(containerWriter, bool, reflect.Value)

	DotResults() []*dot.Result
}

type resultOptions struct {
	// If set, this is the name of the associated result value.
	//
	// For Result Objects, name:".." tags on fields override this.
	Name  string
	Group string
	As    []any
}

var _ func(reflect.Type, resultOptions) (result, error) = newResult

var (
	_ result = (*valueResult)(nil)
	_ result = (*providedObject)(nil)
	_ result = (*valueResultGroup)(nil)
)

// newResult builds a result from the given typ.
func newResult(typ reflect.Type, opts resultOptions) (result, error) {
	switch {
	case IsRequired(typ) ||
		(typ.Kind() == reflect.Pointer && IsRequired(typ.Elem())) ||
		embedsType(typ, _requiredPtrType):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"expected provided struct, but got required struct: %v embeds a dig.Required", typ), nil)

	case isError(typ):
		return nil, newErrInvalidInput("cannot return an error here, return it from the constructor instead", nil)

	case IsProvided(typ):
		return newProvidedObject(typ, opts)

	case embedsType(typ, _providedPtrType):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"provided struct cannot embed *dig.Provided, embed dig.Provided instead: %v embeds *dig.Provided", typ), nil)

	case typ.Kind() == reflect.Pointer && IsProvided(typ.Elem()):
		return nil, newErrInvalidInput(fmt.Sprintf(
			"cannot use pointer as provided struct, use a value instead: %v is a pointer to a struct that embeds dig.Provided", typ), nil)

	case opts.Group != "":
		g, err := parseGroupString(opts.Group)
		if err != nil {
			return nil, newErrInvalidInput(
				fmt.Sprintf("cannot parse group tag value %q", opts.Group), err)
		}
		rg := valueResultGroup{ValueType: typ, GroupName: g.Name, Flatten: g.Flatten}
		if len(opts.As) > 0 {
			var asTypes []reflect.Type
			for _, as := range opts.As {
				ifaceType := reflect.TypeOf(as).Elem() // TODO validate nil
				if ifaceType == typ {
					continue // ignore
				}
				if !typ.Implements(ifaceType) {
					return nil, newErrInvalidInput(
						fmt.Sprintf("invalid dig.As: %v does not implement %v", typ, ifaceType), nil)
				}
				asTypes = append(asTypes, ifaceType)
			}
			if len(asTypes) > 0 {
				rg.ValueType = asTypes[0]
				rg.As = asTypes[1:]
			}
		}
		if g.Passive {
			return nil, newErrInvalidInput(fmt.Sprintf(
				"cannot use group struct tag with passive in provided struct: passive was incorrectly used in group %q", g.Name), nil)
		}
		if g.Flatten {
			if typ.Kind() != reflect.Slice {
				return nil, newErrInvalidInput(fmt.Sprintf(
					"in provided struct, only slice field can use group struct tag with flatten: %v is not a slice", typ), nil)
			}
			rg.ValueType = rg.ValueType.Elem()
		}
		return rg, nil
	default:
		return newValueResult(typ, opts)
	}
}
