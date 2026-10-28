package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// valueResultGroup is a value produced by a constructor that is part of a
// result group.
type valueResultGroup struct {
	// groupName is the name of the group as specified in the `group:".."` tag.
	groupName string

	// valueType is group value type or slice type which
	// element type is group value type.
	valueType reflect.Type

	// flatten indicates elements of a value are to be injected individually,
	// instead of as a group. Requires the value's slice to be a group. If set,
	// valueType is the element type rather than the group (slice) type.
	flatten bool
}

var _ func(reflect.StructField) (valueResultGroup, error) = newValueResultGroup

func (rg valueResultGroup) Provides() []resourceType {
	return []resourceType{rg.key()}
}

func (rg valueResultGroup) StoreValue(w valueWriter, decorated bool, v reflect.Value) {
	if decorated {
		w.submitDecoratedGroup(rg.key(), v)
		return
	}

	if !rg.flatten {
		for _, key := range rg.Provides() {
			w.submitGroupValue(key, v)
		}
		return
	}

	for i := range v.Len() {
		w.submitGroupValue(rg.key(), v.Index(i))
	}
}

func (rg valueResultGroup) DotResults() []*dot.Result {
	return []*dot.Result{
		{
			Node: &dot.Node{
				ValueType: rg.valueType,
				GroupName: rg.groupName,
			},
		},
	}
}

func newValueResultGroup(sf reflect.StructField) (valueResultGroup, error) {
	g, err := parseGroupString(sf.Tag.Get(_groupTag))
	if err != nil {
		return valueResultGroup{}, err
	}

	nameTag := sf.Tag.Get(_nameTag)
	optionalTag, _ := isFieldOptional(sf)

	switch {
	case g.flatten && sf.Type.Kind() != reflect.Slice:
		return valueResultGroup{}, newErrInvalidInput(fmt.Sprintf(
			"in return-struct, only slice field can use group-tag with flatten: "+
				"field %q (%v) is not a slice", sf.Name, sf.Type), nil)

	case g.passive:
		return valueResultGroup{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use group-tag with passive in return-struct: passive was incorrectly used in group %q",
			g.name), nil)

	case nameTag != "":
		return valueResultGroup{}, newErrInvalidInput(fmt.Sprintf(
			"cannot use name-tag with group-tag: name-tag %q used with group-tag %q",
			nameTag, g.name), nil)

	case optionalTag:
		return valueResultGroup{}, newErrInvalidInput(
			"cannot use optional-tag with group-tag", nil)
	}

	valueType := sf.Type
	if g.flatten {
		valueType = valueType.Elem()
	}

	return valueResultGroup{
		groupName: g.name,
		flatten:   g.flatten,
		valueType: valueType,
	}, nil
}

func (rg valueResultGroup) key() resourceType {
	return resourceType{
		valueType: rg.valueType,
		groupName: rg.groupName,
	}
}
