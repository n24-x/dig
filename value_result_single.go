package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// valueResult is an explicit value produced by a constructor, optionally
// with a name and/or extra As types.
type valueResult struct {
	valueName string
	valueType reflect.Type

	asTypes []reflect.Type
}

var _ func(reflect.Type, resultOptions) (valueResult, error) = newValueResult

func (r valueResult) Provides() []resourceType {
	keys := []resourceType{
		r.key(),
	}
	for _, typ := range r.asTypes {
		keys = append(keys, resourceType{
			valueType: typ,
			valueName: r.valueName,
		})
	}
	return keys
}

func (r valueResult) Extract(w valueWriter, v reflect.Value) {
	for _, key := range r.Provides() {
		w.setValue(key, v)
	}
}

func (r valueResult) DotResults() []*dot.Result {
	dotResults := []*dot.Result{
		{
			Node: &dot.Node{
				ValueType: r.valueType,
				ValueName: r.valueName,
			},
		},
	}
	if len(r.asTypes) > 0 {
		for _, typ := range r.asTypes {
			dotResults = append(dotResults, &dot.Result{
				Node: &dot.Node{
					ValueType: typ,
					ValueName: r.valueName,
				},
			})
		}
	}
	return dotResults
}

func newValueResult(typ reflect.Type, opts resultOptions) (valueResult, error) {
	r := valueResult{
		valueType: typ,
		valueName: opts.nameTag,
	}

	var asTypes []reflect.Type
	for _, as := range opts.asTypes {
		ifaceType := reflect.TypeOf(as).Elem() // as is validated
		if ifaceType == typ {
			// Special case:
			//   c.Resource(func() io.Reader, As(new(io.Reader)))
			// Ignore instead of erroring out.
			continue
		}
		if !typ.Implements(ifaceType) {
			return r, newErrInvalidInput(
				fmt.Sprintf("invalid dig.As(%v): does not implement %v",
					typ, ifaceType), nil)
		}
		asTypes = append(asTypes, ifaceType)
	}

	if len(asTypes) == 0 {
		return r, nil
	}

	return valueResult{
		valueType: asTypes[0],
		valueName: opts.nameTag,
		asTypes:   asTypes[1:],
	}, nil
}

func (r valueResult) key() resourceType {
	return resourceType{
		valueType: r.valueType,
		valueName: r.valueName,
	}
}
