package dig

import (
	"fmt"

	"github.com/n24-x/dag-go"
)

// nodeColor is a node's mark during the walk.
//
// It is a distinct type rather than a bare uint8 so that assigning anything
// else to ctorWalk.state is a compile error.
type nodeColor uint8

const (
	_white nodeColor = iota // undiscovered: not visited yet
	_gray                   // on the current DFS path
	_black                  // fully processed and already emitted
)

// ctorWalk computes the order in which constructors have to run, starting from
// a function's params.
//
// It walks the graph with a three-colour depth-first search: white nodes are
// undiscovered, gray nodes sit on the current path, black nodes are done and
// emitted. Required edges come from the dag; optional edges are followed too,
// but only when somebody provides the key, so an optional dep whose key nobody
// provides contributes no node and the param keeps its zero value.
//
// An optional edge is followed only when its provider can still run before the
// consumer. If the provider's required closure leads back to the consumer, the
// two intents contradict each other and the walk fails: silently handing the
// consumer a zero value would hide a real mistake in the wiring. Required edges
// have no such escape hatch, but dag.Add has already rejected cycles over them,
// so a back edge there is an invariant violation.
//
// Nodes are emitted on the way out, so the result is a post-order and every
// dependency precedes its consumer.
type ctorWalk struct {
	sp    *Scope
	state []nodeColor // indexed by NodeID; 0 is unused

	// order holds the resolved nodes in run order, not their NodeIDs: the walk
	// has to look each one up to reach its params anyway, so doing the cast here
	// keeps callers from repeating it.
	order []*constructorNode

	// missing collects required keys that nobody provides, in the order they
	// were first seen. seen keeps that list free of duplicates.
	missing []resourceType
	seen    map[resourceType]struct{} // TODO go1.28 change to container/set
}

func newCtorWalk(sp *Scope) *ctorWalk {
	count := sp.dag.Count()
	return &ctorWalk{
		sp:    sp,
		state: make([]nodeColor, count+1),
		order: make([]*constructorNode, 0, count),
		seen:  make(map[resourceType]struct{}),
	}
}

// visit emits id after everything it depends on.
func (w *ctorWalk) visit(nodeID dag.NodeID) error {
	switch w.state[nodeID] {
	case _black: // already emitted, nothing to do
		return nil
	case _gray: // reached again while still on the path
		// This error only for developers, should never be encountered by users.
		return fmt.Errorf("dig: internal error: circular dependency detected while "+
			"ordering constructors at node %d", nodeID)
	case _white:
		// Fresh node: fall through to the walk below.
	}
	w.state[nodeID] = _gray

	node, ok := w.sp.dag.At(nodeID).(*constructorNode)
	if !ok {
		// This error only for developers, should never be encountered by users.
		return fmt.Errorf("dig: internal error: DAG node %d is not a constructor", nodeID)
	}

	// Required deps: dag edges. A required dep that nobody provides has no
	// edge and would otherwise be skipped in silence, so check it here.
	for _, key := range node.Requires() {
		if len(w.sp.dag.Provide(key)) == 0 {
			w.addMissing(key)
		}
	}
	for _, dep := range w.sp.dag.OutNeighbors(nodeID) {
		if err := w.visit(dep); err != nil {
			return err
		}
	}

	// Optional deps: no edge in the dag, so resolve the providers by hand.
	// If the provider would need this node to have run first, the two cannot
	// both be satisfied; report it rather than silently handing the consumer a
	// zero value.
	for _, key := range node.ParamList().Requires().Optional {
		for _, depNodeID := range w.sp.dag.Provide(key) {
			if w.sp.requiredReaches(depNodeID, nodeID) {
				return fmt.Errorf(
					"dig: constructor %v optionally needs %v, but its provider %v "+
						"requires this constructor: resolve the cycle",
					node, key, w.sp.dag.At(depNodeID))
			}
			if err := w.visit(depNodeID); err != nil {
				return err
			}
		}
	}

	w.state[nodeID] = _black
	w.order = append(w.order, node)
	return nil
}

// addMissing records key once.
func (w *ctorWalk) addMissing(key resourceType) {
	if _, ok := w.seen[key]; ok {
		return
	}
	w.seen[key] = struct{}{}
	w.missing = append(w.missing, key)
}

// err reports the missing providers collected so far, if any.
func (w *ctorWalk) err() error {
	if len(w.missing) == 0 {
		return nil
	}
	// TODO handle error
	return &dag.MissingProviderError[resourceType]{Resources: w.missing}
}

// requiredReaches reports whether target is reachable from start by following
// required edges only.
//
// It is used to decide whether an optional edge may be followed: if the
// provider would need the consumer to have run first, the two contradict each
// other and the graph is rejected.
//
// TODO: this walks a subgraph per optional edge. Fine for the graph sizes dig
// sees; revisit if that ever changes.
func (sp *Scope) requiredReaches(start, target dag.NodeID) bool {
	if start == target {
		return true
	}
	seen := map[dag.NodeID]bool{start: true} // TODO go1.28 change to container/set
	stack := []dag.NodeID{start}
	for len(stack) > 0 {
		u := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, v := range sp.dag.OutNeighbors(u) {
			if v == target {
				return true
			}
			if !seen[v] {
				seen[v] = true
				stack = append(stack, v)
			}
		}
	}
	return false
}
