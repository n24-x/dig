package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// valueResult is an explicit value produced by a constructor, optionally
// with a name and/or extra As types.
type valueResult struct {
	ValueName string
	ValueType reflect.Type

	As []reflect.Type
}

var _ func(reflect.Type, resultOptions) (valueResult, error) = newValueResult

func (r valueResult) Provides() []resourceType {
	keys := []resourceType{
		r.key(),
	}
	for _, typ := range r.As {
		keys = append(keys, resourceType{
			valueType: typ,
			valueName: r.ValueName,
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
				ValueType: r.ValueType,
				ValueName: r.ValueName,
			},
		},
	}
	if len(r.As) > 0 {
		for _, typ := range r.As {
			dotResults = append(dotResults, &dot.Result{
				Node: &dot.Node{
					ValueType: typ,
					ValueName: r.ValueName,
				},
			})
		}
	}
	return dotResults
}

func newValueResult(typ reflect.Type, opts resultOptions) (valueResult, error) {
	r := valueResult{
		ValueType: typ,
		ValueName: opts.ValueName,
	}

	var asTypes []reflect.Type
	for _, as := range opts.As {
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
		ValueType: asTypes[0],
		ValueName: opts.ValueName,
		As:        asTypes[1:],
	}, nil
}

func (r valueResult) key() resourceType {
	return resourceType{
		valueType: r.ValueType,
		valueName: r.ValueName,
	}
}
