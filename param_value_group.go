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

	// Soft is used to denote a soft dependency between this param and its
	// constructors, if it's true its constructors are only called if they
	// provide another value requested in the graph
	Soft bool
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
// Soft groups are not reported: graph edges are "must build" edges, and a
// soft group must not force its providers to run. It is populated with
// whatever was built for another reason, falling back to an empty slice.
func (pg valueParamGroup) Requires() []resourceKey {
	if pg.Soft {
		return nil
	}
	return []resourceKey{
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

func newValueParamGroup(f reflect.StructField) (valueParamGroup, error) {
	g, err := parseGroupString(f.Tag.Get(_groupTag))
	if err != nil {
		return valueParamGroup{}, err
	}

	pg := valueParamGroup{
		GroupName: g.Name,
		ValueType: f.Type,
		Soft:      g.Soft,
	}

	named := f.Tag.Get(_nameTag)
	optional, _ := isFieldOptional(f)
	switch {
	case f.Type.Kind() != reflect.Slice:
		return pg, newErrInvalidInput(
			fmt.Sprintf(
				"value groups may be consumed as slices only: field %q (%v) is not a slice",
				f.Name,
				f.Type,
			), nil)
	case g.Flatten:
		return pg, newErrInvalidInput(
			fmt.Sprintf("cannot use flatten in parameter value groups: field %q (%v) specifies flatten", f.Name, f.Type), nil)
	case named != "":
		return pg, newErrInvalidInput(
			fmt.Sprintf("cannot use named values with value groups: name:%q requested with group:%q", named, pg.GroupName), nil)
	case optional:
		return pg, newErrInvalidInput("value groups cannot be optional", nil)
	}
	return pg, nil
}
