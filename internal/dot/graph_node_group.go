package dot

import (
	"fmt"
	"reflect"
)

// Group is a group node in the graph.
//
// In a DOT image, each (type, group name) pair is represented by exactly one
// diamond node, and that node connects to the Result nodes belonging to the group.
type Group struct {
	// ValueType is the type of values in the group.
	ValueType reflect.Type
	Name      string
	Results   []*Result
	ErrorType ErrorType
}

func (g *Group) nodeKey() nodeKey {
	return nodeKey{valueType: g.ValueType, groupName: g.Name}
}

// NewGroup creates a new group with information in the groupKey.
func NewGroup(k nodeKey) *Group {
	return &Group{
		ValueType: k.valueType,
		Name:      k.groupName,
	}
}

func (g *Group) removeResult(r *Result) {
	var pruned []*Result
	for _, rg := range g.Results {
		if r.IndexInGroup != rg.IndexInGroup {
			pruned = append(pruned, rg)
		}
	}
	g.Results = pruned
}

// DotString returns the identifier used for this node in DOT output.
func (g *Group) DotString() string {
	return fmt.Sprintf("[type=%v group=%v]", g.ValueType.String(), g.Name)
}

// DotAttributes composes and returns a string of the Group node's attributes.
func (g *Group) DotAttributes() string {
	attr := fmt.Sprintf(
		`shape=diamond label=<%v<BR /><FONT POINT-SIZE="10">Group: %v</FONT>>`,
		g.ValueType,
		g.Name,
	)
	if g.ErrorType != noError {
		attr += " color=" + g.ErrorType.Color()
	}
	return attr
}
