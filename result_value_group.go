package dig

import "reflect"

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
