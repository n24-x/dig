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

// DecorationKeys reports the key this result takes over as a decorator.
//
// A group is decorated as a whole, and readers look it up by its element type,
// so that is where the decoration has to land. A plain `T group:"g"` field is
// one member of the group, not the group, so it cannot stand for it.
func (rg valueResultGroup) DecorationKeys() ([]resourceType, error) {
	if !rg.flatten && rg.valueType.Kind() != reflect.Slice {
		return nil, newErrInvalidInput(fmt.Sprintf(
			"decorating group %q needs a slice result holding the new members, "+
				"not a single one: %v is not a slice", rg.groupName, rg.valueType), nil)
	}
	return []resourceType{rg.decorationKey()}, nil
}

// decorationKey is the key a decoration of this group is written under.
//
// Provides names the raw value the constructor returns, which for a plain
// `[]T group:"g"` field is a `[]T` member of a different group; readers ask for
// the group by its element type, so the decoration has to go there instead.
// flatten already keys that way, because its valueType is the element type.
func (rg valueResultGroup) decorationKey() resourceType {
	if rg.flatten {
		return rg.key()
	}
	return resourceType{valueType: rg.valueType.Elem(), groupName: rg.groupName}
}

func (rg valueResultGroup) StoreValue(w valueWriter, decorated bool, v reflect.Value) {
	if decorated {
		// The decorator returns the whole new group, so it goes in as one value
		// under the key readers ask for. Registration rejected the results that
		// cannot be spread back, so decorationKey cannot panic here.
		w.submitDecoratedGroup(rg.decorationKey(), v)
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
