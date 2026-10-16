package dig

import (
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// valueResultGroup is a value produced by a constructor that is part of a
// result group.
type valueResultGroup struct {
	// Name of the group as specified in the `group:".."` tag.
	GroupName string

	// ValueType is the element type contributed to the group.
	ValueType reflect.Type

	// Indicates elements of a value are to be injected individually, instead of
	// as a group. Requires the value's slice to be a group. If set, ValueType is
	// the element type rather than the group (slice) type.
	Flatten bool

	As []reflect.Type
}

func (rg valueResultGroup) Provides() []resourceKey {
	keys := []resourceKey{
		{
			valueType: rg.ValueType,
			groupName: rg.GroupName,
		},
	}
	for _, typ := range rg.As {
		keys = append(keys, resourceKey{
			valueType: typ,
			groupName: rg.GroupName,
		})
	}
	return keys
}

func (rg valueResultGroup) DotResults() []*dot.Result {
	dotResults := []*dot.Result{
		{
			Node: &dot.Node{
				ValueType: rg.ValueType,
				GroupName: rg.GroupName,
			},
		},
	}

	if len(rg.As) > 0 {
		for _, typ := range rg.As {
			dotResults = append(dotResults,
				&dot.Result{
					Node: &dot.Node{
						ValueType: typ,
						GroupName: rg.GroupName,
					},
				},
			)
		}
	}

	return dotResults
}
