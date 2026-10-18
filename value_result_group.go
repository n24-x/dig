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

func (rg valueResultGroup) Provides() []resourceID {
	keys := []resourceID{
		{
			valueType: rg.ValueType,
			groupName: rg.GroupName,
		},
	}
	for _, typ := range rg.As {
		keys = append(keys, resourceID{
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

	nameTag := sf.Tag.Get(_nameTag)
	optionalTag, _ := isFieldOptional(sf)

	switch {
	case g.Flatten && sf.Type.Kind() != reflect.Slice:
		return rg, newErrInvalidInput(fmt.Sprintf(
			"in return-struct, only slice field can use group-tag with flatten: "+
				"field %q (%v) is not a slice", sf.Name, sf.Type), nil)

	case g.Passive:
		return rg, newErrInvalidInput(fmt.Sprintf(
			"cannot use group-tag with passive in return-struct: passive was incorrectly used in group %q",
			rg.GroupName), nil)

	case nameTag != "":
		return rg, newErrInvalidInput(fmt.Sprintf(
			"cannot use name-tag with group-tag: name-tag %q used with group-tag %q",
			nameTag, rg.GroupName), nil)

	case optionalTag:
		return rg, newErrInvalidInput(
			"cannot use optional-tag with group-tag", nil)
	}

	if g.Flatten {
		rg.ValueType = sf.Type.Elem()
	}

	return rg, nil
}
