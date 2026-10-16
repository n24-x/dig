package dig

import (
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

func (r valueResult) Provides() []resourceKey {
	keys := []resourceKey{
		{
			valueType: r.ValueType,
			valueName: r.ValueName,
		},
	}
	for _, typ := range r.As {
		keys = append(keys, resourceKey{
			valueType: typ,
			valueName: r.ValueName,
		})
	}
	return keys
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
