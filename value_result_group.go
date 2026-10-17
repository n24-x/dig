package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// valueResultGroup is a value produced by a constructor that is part of a
// result group.
type valueResultGroup struct {
	// Name of the group as specified in the `group:".."` tag.
	GroupName string

	// ValueType is the value or value element type contributed to the group.
	ValueType reflect.Type

	// Indicates elements of a value are to be injected individually, instead of
	// as a group. Requires the value's slice to be a group. If set, ValueType is
	// the element type rather than the group (slice) type.
	Flatten bool

	As []reflect.Type
}

var _ func(reflect.StructField) (valueResultGroup, error) = newValueResultGroup

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

func newValueResultGroup(sf reflect.StructField) (valueResultGroup, error) {
	g, err := parseGroupString(sf.Tag.Get(_groupTag))
	if err != nil {
		return valueResultGroup{}, err
	}

	rg := valueResultGroup{
		GroupName: g.Name,
		Flatten:   g.Flatten,
		ValueType: sf.Type,
	}

	named := sf.Tag.Get(_nameTag)
	optional, _ := isFieldOptional(sf)

	switch {
	case g.Flatten && sf.Type.Kind() != reflect.Slice:
		return rg, newErrInvalidInput(fmt.Sprintf(
			"flatten must apply to slices: field %q (%v) is not a slice", sf.Name, sf.Type), nil)

	case g.Soft:
		return rg, newErrInvalidInput(fmt.Sprintf(
			"cannot use soft in result groups struct tag: soft was incorrectly used in group %q",
			rg.GroupName), nil)

	case named != "":
		return rg, newErrInvalidInput(fmt.Sprintf(
			"cannot use named values with value groups: name:%q provided with group:%q",
			named, rg.GroupName), nil)

	case optional:
		return rg, newErrInvalidInput(
			"cannot use optional struct tag with result groups", nil)
	}

	if g.Flatten {
		rg.ValueType = sf.Type.Elem()
	}

	return rg, nil
}
