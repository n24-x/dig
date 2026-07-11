// Copyright (c) 2019 Uber Technologies, Inc.
// Copyright (c) 2026 k2 <skrik2@outlook.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package dot

import (
	"fmt"
	"reflect"
)

// ConstructorID uniquely identifies a constructor by its function pointer.
// It is used as the key for constructor lookups in the dependency graph.
type ConstructorID uintptr

// Constructor encodes a constructor provided to the container for the DOT graph.
type Constructor struct {
	ID      ConstructorID // Unique constructor identifier.
	Name    string        // Constructor name, such as "NewHandler".
	Package string        // Package path, such as "github.com/n24-x/dig"
	File    string        // Source file path.
	Line    int           // Source line number.

	// Params is the list of regular input dependencies of this constructor.
	// Each Param represents a single dependency matched by type and optionally by name.
	Params []*Param

	// GroupParams is the list of value group input dependencies of this constructor.
	// Each Group represents a collection of values sharing the same type and group name.
	GroupParams []*Group

	// Results is the list of values produced by this constructor.
	Results []*Result

	ErrorType ErrorType // Constructor error state.
}

func (c *Constructor) removeParam(k nodeKey) {
	var pruned []*Param
	for _, p := range c.Params {
		if k != p.nodeKey() {
			pruned = append(pruned, p)
		}
	}
	c.Params = pruned
}

// Param is a parameter node in the graph. Parameters are the input to constructors.
type Param struct {
	*Node

	Optional bool
}

// String implements fmt.Stringer for Param.
func (p *Param) String() string {
	if p.Name != "" {
		return fmt.Sprintf("%v[name=%v]", p.Type.String(), p.Name)
	}
	return p.Type.String()
}

// Group is a group node in the graph.
//
// It is unique for each (type, group name) pair in a [Graph] and can be shared by
// multiple constructors.
type Group struct {
	// Type is the type of values in the group.
	Type      reflect.Type
	Name      string
	Results   []*Result
	ErrorType ErrorType
}

func (g *Group) nodeKey() nodeKey {
	return nodeKey{t: g.Type, group: g.Name}
}

// NewGroup creates a new group with information in the groupKey.
func NewGroup(k nodeKey) *Group {
	return &Group{
		Type: k.t,
		Name: k.group,
	}
}

// TODO(rhang): Avoid linear search to discover group results that should be pruned.
func (g *Group) removeResult(r *Result) {
	var pruned []*Result
	for _, rg := range g.Results {
		if r.GroupIndex != rg.GroupIndex {
			pruned = append(pruned, rg)
		}
	}
	g.Results = pruned
}

// Attributes composes and returns a string of the Group node's attributes.
func (g *Group) Attributes() string {
	attr := fmt.Sprintf(`shape=diamond label=<%v<BR /><FONT POINT-SIZE="10">Group: %v</FONT>>`, g.Type, g.Name)
	if g.ErrorType != noError {
		attr += " color=" + g.ErrorType.Color()
	}
	return attr
}

// String implements fmt.Stringer for Group.
func (g *Group) String() string {
	return fmt.Sprintf("[type=%v group=%v]", g.Type.String(), g.Name)
}

// Result is a result node in the graph. It represents a value produced by a
// constructor but does not store the runtime value itself. Grouped results are
// additionally referenced by their corresponding [Group].
type Result struct {
	*Node

	// GroupIndex is added to differentiate grouped values from one another.
	// Since grouped values have the same type and group, their Node / string
	// representations are the same so we need indices to uniquely identify
	// the values.
	GroupIndex int
}

// String implements fmt.Stringer for Result.
func (r *Result) String() string {
	switch {
	case r.Name != "":
		return fmt.Sprintf("%v[name=%v]", r.Type.String(), r.Name)
	case r.Group != "":
		return fmt.Sprintf("%v[group=%v]%v", r.Type.String(), r.Group, r.GroupIndex)
	default:
		return r.Type.String()
	}
}

// Attributes composes and returns a string of the Result node's attributes.
func (r *Result) Attributes() string {
	switch {
	case r.Name != "":
		return fmt.Sprintf(`label=<%v<BR /><FONT POINT-SIZE="10">Name: %v</FONT>>`, r.Type, r.Name)
	case r.Group != "":
		return fmt.Sprintf(`label=<%v<BR /><FONT POINT-SIZE="10">Group: %v</FONT>>`, r.Type, r.Group)
	default:
		return fmt.Sprintf(`label=<%v>`, r.Type)
	}
}
