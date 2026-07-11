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

import "reflect"

// Graph is the DOT-format graph in a Container.
type Graph struct {
	Constructors   []*Constructor
	constructorMap map[ConstructorID]*Constructor

	Groups   []*Group
	groupMap map[nodeKey]*Group

	consumers map[nodeKey][]*Constructor

	Failed *FailedNodes
}

// FailedNodes is the nodes that failed in the graph.
type FailedNodes struct {
	// RootCauses is a list of the point of failures. They are the root causes
	// of failed invokes and can be either missing types (not provided) or
	// error types (error providing).
	RootCauses []*Result

	// TransitiveFailures is the list of nodes that failed to build due to
	// missing/failed dependencies.
	TransitiveFailures []*Result

	// constructors is a collection of failed constructors IDs that are populated as the graph is
	// traversed for errors.
	constructors map[ConstructorID]struct{}

	// groups is a collection of failed groupKeys that is populated as the graph is traversed
	// for errors.
	groups map[nodeKey]struct{}
}

// NewGraph creates an empty graph.
func NewGraph() *Graph {
	return &Graph{
		constructorMap: make(map[ConstructorID]*Constructor),
		groupMap:       make(map[nodeKey]*Group),
		consumers:      make(map[nodeKey][]*Constructor),
		Failed: &FailedNodes{
			constructors: make(map[ConstructorID]struct{}),
			groups:       make(map[nodeKey]struct{}),
		},
	}
}

// AddConstructor adds the constructor with paramList and resultList into the graph.
func (dg *Graph) AddConstructor(c *Constructor, paramList []*Param, resultList []*Result) {
	var (
		params      []*Param
		groupParams []*Group
	)

	// Loop through the paramList to separate them into regular params and
	// grouped params. For grouped params, we use getGroup to find the actual
	// group.
	for _, param := range paramList {
		if param.Group == "" {
			// Not a value group
			params = append(params, param)
			continue
		}

		key := nodeKey{t: param.Type.Elem(), group: param.Group}
		group := dg.getGroup(key)
		groupParams = append(groupParams, group)
	}

	for _, result := range resultList {
		// If the result is a grouped value, we want to update its GroupIndex
		// and add it to the Group.
		if result.Group != "" {
			dg.addToGroup(result)
		}
	}

	c.Params = params
	c.GroupParams = groupParams
	c.Results = resultList

	// Track which constructors consume a parameter.
	for _, p := range paramList {
		key := p.nodeKey()
		dg.consumers[key] = append(dg.consumers[key], c)
	}

	dg.Constructors = append(dg.Constructors, c)
	dg.constructorMap[c.ID] = c
}

// getGroup finds the group by nodeKey from the graph. If it is not available,
// a new group is created and returned.
func (dg *Graph) getGroup(key nodeKey) *Group {
	g, ok := dg.groupMap[key]
	if !ok {
		g = NewGroup(key)
		dg.groupMap[key] = g
		dg.Groups = append(dg.Groups, g)
	}
	return g
}

// addToGroup adds a newly provided grouped result to the appropriate group.
func (dg *Graph) addToGroup(r *Result) {
	key := nodeKey{t: r.Type, group: r.Group}
	group := dg.getGroup(key)

	r.GroupIndex = len(group.Results)
	group.Results = append(group.Results, r)
}

// AddMissingNodes adds missing nodes to the list of failed Results in the graph.
func (dg *Graph) AddMissingNodes(results []*Result) {
	// The failure(s) are root causes if there are no other failures.
	isRootCause := len(dg.Failed.RootCauses) == 0

	for _, r := range results {
		dg.failNode(r, isRootCause)
	}
}

func (dg *Graph) failNode(r *Result, isRootCause bool) {
	if isRootCause {
		dg.addRootCause(r)
	} else {
		dg.addTransitiveFailure(r)
	}
}

func (dg *Graph) addRootCause(r *Result) {
	dg.Failed.RootCauses = append(dg.Failed.RootCauses, r)
}

func (dg *Graph) addTransitiveFailure(r *Result) {
	dg.Failed.TransitiveFailures = append(dg.Failed.TransitiveFailures, r)

}

// FailNodes adds results to the list of failed Results in the graph, and
// updates the state of the constructor with the given id accordingly.
func (dg *Graph) FailNodes(results []*Result, id ConstructorID) {
	// This failure is the root cause if there are no other failures.
	isRootCause := len(dg.Failed.RootCauses) == 0
	dg.Failed.constructors[id] = struct{}{}

	for _, r := range results {
		dg.failNode(r, isRootCause)
	}

	if c, ok := dg.constructorMap[id]; ok {
		if isRootCause {
			c.ErrorType = rootCause
		} else {
			c.ErrorType = transitiveFailure
		}
	}
}

// FailGroupNodes finds and adds the failed grouped nodes to the list of failed
// Results in the graph, and updates the state of the group and constructor
// with the given id accordingly.
func (dg *Graph) FailGroupNodes(name string, t reflect.Type, id ConstructorID) {
	// This failure is the root cause if there are no other failures.
	isRootCause := len(dg.Failed.RootCauses) == 0

	key := nodeKey{t: t, group: name}
	group := dg.getGroup(key)

	// If the constructor does not exist it cannot be failed.
	if _, ok := dg.constructorMap[id]; !ok {
		return
	}

	// Track which constructors and groups have failed.
	dg.Failed.constructors[id] = struct{}{}
	dg.Failed.groups[key] = struct{}{}

	for _, r := range dg.constructorMap[id].Results {
		if r.Type == t && r.Group == name {
			dg.failNode(r, isRootCause)
		}
	}

	if c, ok := dg.constructorMap[id]; ok {
		if isRootCause {
			group.ErrorType = rootCause
			c.ErrorType = rootCause
		} else {
			group.ErrorType = transitiveFailure
			c.ErrorType = transitiveFailure
		}
	}
}

// PruneSuccess removes elements from the graph that do not have failed results.
// Removing elements that do not have failing results makes the graph easier to debug,
// since non-failing nodes and edges can clutter the graph and don't help the user debug.
func (dg *Graph) PruneSuccess() {
	dg.pruneCtors(dg.Failed.constructors)
	dg.pruneGroups(dg.Failed.groups)
}

// pruneCtors removes constructors from the graph that do not have failing Results.
func (dg *Graph) pruneCtors(failed map[ConstructorID]struct{}) {
	var pruned []*Constructor
	for _, c := range dg.Constructors {
		if _, ok := failed[c.ID]; ok {
			pruned = append(pruned, c)
			continue
		}
		// If a constructor is deleted, the constructor's stale result references need to
		// be removed from that result's Group and/or consuming constructor.
		dg.pruneCtorParams(c, dg.consumers)
		dg.pruneGroupResults(c, dg.groupMap)
		delete(dg.constructorMap, c.ID)
	}
	dg.Constructors = pruned
}

// pruneCtorParams removes results of the constructor argument that are still referenced in the
// Params of constructors that consume those results. If the results in the constructor are found
// in the params of a consuming constructor that result should be removed.
func (dg *Graph) pruneCtorParams(c *Constructor, consumers map[nodeKey][]*Constructor) {
	for _, r := range c.Results {
		for _, ctor := range consumers[r.nodeKey()] {
			ctor.removeParam(r.nodeKey())
		}
	}
}

// pruneGroupResults removes results of the constructor argument that are still referenced in
// the Group object that contains that result. If a group no longer exists references to that
// should should be removed.
func (dg *Graph) pruneGroupResults(c *Constructor, groups map[nodeKey]*Group) {
	for _, r := range c.Results {
		key := r.nodeKey()
		if key.group == "" {
			continue
		}

		g, ok := groups[key]
		if ok {
			g.removeResult(r)
		}
	}
}

// pruneGroups removes groups from the graph that do not have failing results.
func (dg *Graph) pruneGroups(failed map[nodeKey]struct{}) {
	var pruned []*Group
	for _, g := range dg.Groups {
		key := g.nodeKey()
		if _, ok := failed[key]; ok {
			pruned = append(pruned, g)
			continue
		}
		delete(dg.groupMap, key)
	}
	dg.Groups = pruned

	dg.pruneCtorGroupParams(dg.groupMap)
}

// pruneCtorGroupParams removes constructor results that are still referenced in the GroupParams of
// constructors that consume those results.
func (dg *Graph) pruneCtorGroupParams(groups map[nodeKey]*Group) {
	for _, c := range dg.Constructors {
		var pruned []*Group
		for _, gp := range c.GroupParams {
			key := gp.nodeKey()
			if _, ok := groups[key]; ok {
				pruned = append(pruned, gp)
			}
		}
		c.GroupParams = pruned
	}
}
