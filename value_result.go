package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// The result interface represents a result produced by a constructor.
type result interface {
	Provides() []resourceType

	Extract(w valueWriter, v reflect.Value)

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

	case isError(typ):
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

	case opts.GroupName != "":
		g, err := parseGroupString(opts.GroupName)
		if err != nil {
			return nil, newErrInvalidInput(
				fmt.Sprintf("cannot parse group-tag value %q", opts.GroupName), err)
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
				"cannot use group-tag with passive in return-struct:"+
					" passive was incorrectly used in group %q", g.Name), nil)
		}
		if g.Flatten {
			if typ.Kind() != reflect.Slice {
				return nil, newErrInvalidInput(fmt.Sprintf(
					"in return-struct, only slice field can use group-tag with flatten:"+
						" %v is not a slice", typ), nil)
			}
			rg.ValueType = rg.ValueType.Elem()
		}
		return rg, nil
	default:
		return newValueResult(typ, opts)
	}
}
