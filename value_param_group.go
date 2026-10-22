package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

// valueParamGroup is a param which produces a slice of values with the same
// group name.
type valueParamGroup struct {
	// Name of the group as specified in the `group:".."` tag.
	GroupName string

	// ValueType of the slice.
	ValueType reflect.Type

	// Passive is used to denote a passive dependency between this param and its
	// constructors: if true, its constructors are only called if they provide
	// another value requested in the graph
	Passive bool
}

var _ func(reflect.StructField) (valueParamGroup, error) = newValueParamGroup

func (pg valueParamGroup) String() string {
	// io.Reader[group="foo"] refers to a group of io.Readers called 'foo'
	return fmt.Sprintf("%v[group=%q]", pg.ValueType.Elem(), pg.GroupName)
}

// Requires reports the resources this param depends on: the group itself,
// keyed by the element type of the slice. Group providers emit their results
// under the same key, so all providers of a group map to one resource.
//
// Passive groups are not reported: graph edges are "must build" edges, and a
// passive group must not force its providers to run. It is populated with
// whatever was built for another reason, falling back to an empty slice.
func (pg valueParamGroup) Requires() []resourceType {
	if pg.Passive {
		return nil
	}
	return []resourceType{
		{
			valueType: pg.ValueType.Elem(),
			groupName: pg.GroupName,
		},
	}
}

func (pg valueParamGroup) DotParams() []*dot.Param {
	return []*dot.Param{
		{
			Node: &dot.Node{
				ValueType: pg.ValueType,
				GroupName: pg.GroupName,
			},
		},
	}
}

func newValueParamGroup(sf reflect.StructField) (valueParamGroup, error) {
	g, err := parseGroupString(sf.Tag.Get(_groupTag))
	if err != nil {
		return valueParamGroup{}, err
	}

	nameTag := sf.Tag.Get(_nameTag)
	optionalTag, _ := isFieldOptional(sf)
	switch {
	case sf.Type.Kind() != reflect.Slice:
		return valueParamGroup{}, newErrInvalidInput(
			fmt.Sprintf(
				"in parameter-struct, only slice field can use group-tag: field %q (%v) is not a slice",
				sf.Name,
				sf.Type,
			), nil)
	case g.Flatten:
		return valueParamGroup{}, newErrInvalidInput(
			fmt.Sprintf("cannot use group-tag with flatten in parameter-struct: "+
				"field %q (%v) specifies flatten", sf.Name, sf.Type), nil)
	case nameTag != "":
		return valueParamGroup{}, newErrInvalidInput(
			fmt.Sprintf("cannot use name-tag with group-tag: "+
				"name-tag %q used with group-tag %q", nameTag, g.Name), nil)
	case optionalTag:
		return valueParamGroup{}, newErrInvalidInput("cannot use optional-tag with group-tag", nil)
	}

	return valueParamGroup{
		GroupName: g.Name,
		ValueType: sf.Type,
		Passive:   g.Passive,
	}, nil
}

// TODO
func (po valueParamGroup) Value(sp *Scope) (reflect.Value, error) {
	return reflect.Value{}, nil
}
