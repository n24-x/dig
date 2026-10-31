package dig

import (
	"fmt"

	"github.com/n24-x/dag-go"
)

type nodeColor uint8

const (
	_white nodeColor = iota // undiscovered: absent from the table, not stored
	_gray                   // on the current DFS path
	_black                  // fully processed and already emitted
)

// keyWalk computes the execution order of constructors and decorators from a
// function's params using a three-colour depth-first search (DFS) in post-order.
//
// The walk resolves one key at a time instead of following the dag's edges. A
// decorator is not a dag node, so edges alone cannot see it, and the decorator
// for a key has to be emitted after the constructors that build its input.
//
// Optional deps are followed too, but only when somebody provides the key, so
// an optional dep whose key nobody provides contributes no node and the param
// keeps its zero value. An optional dep has no dag edge, so following one can
// close a loop that dag.Add never saw; that is reported as a wiring mistake
// rather than as the invariant violation a real cycle is.
//
// Every lookup is made through a scope, and that scope is never "the one being
// set up". A key is read through the view of whoever reads it: a constructor's
// params from the scope Resource was called on, a decorator's from the scope it
// decorates in, and the function being set up from the scope Setup ran on. The
// walk keeps no scope of its own, so there is no second view to fall back on by
// accident.
type keyWalk struct {
	state map[runner]nodeColor // one table for both node kinds

	// path is the DFS stack: the nodes currently gray, outermost first. It
	// exists so a cycle can be reported as the sequence that closes it rather
	// than as a single node; the stack is pushed and popped by visit and
	// visitDecorator, so it always mirrors the recursion.
	path []runner

	// order holds the resolved nodes in run order. The walk reaches each one's
	// params anyway, so keeping the nodes here saves callers from looking them
	// up again.
	order []runner

	// missing collects required keys that nobody provides, in the order they
	// were first seen. seen keeps that list free of duplicates.
	missing []resourceType
	seen    map[resourceType]struct{} // TODO go1.28 change to container/set

	// optionalPath holds the optional edges open on the current DFS path. A
	// gray hit is a cycle either way; whether the path came in through one of
	// these is what separates a wiring mistake from a broken graph.
	optionalPath []optionalEdge

	// currentDecorator is the decorator whose params are being resolved right
	// now, or nil when the walk is not inside one.
	//
	// It tells two gray decorators apart. The one we are inside reads the key
	// it takes over on purpose and has to see the raw value; any other gray
	// decorator means two of them need each other's output and neither can go
	// first.
	currentDecorator *decoratorNode
}

// optionalEdge corresponds to optional dependency
type optionalEdge struct {
	// consumer is the node that asked, or nil for the function being set up.
	consumer fmt.Stringer
	key      resourceType
}

// newKeyWalk returns a walk sized for a scope holding count nodes.
func newKeyWalk(count int) *keyWalk {
	return &keyWalk{
		state: make(map[runner]nodeColor, count),
		order: make([]runner, 0, count),
		seen:  make(map[resourceType]struct{}),
	}
}

// resolveRequired resolves a required key read through from.
// If no provider exists, the key is recorded as missing.
func (w *keyWalk) resolveRequired(from *Scope, key resourceType) error {
	found, err := w.resolve(from, key)
	if err != nil {
		return err
	}
	if !found {
		w.addMissing(key)
	}
	return nil
}

// resolveOptional resolves an optional key read through from. If a provider
// exists, it resolves the key and tracks the optional dependency so that a
// cycle involving it can be identified.
func (w *keyWalk) resolveOptional(from *Scope, consumer fmt.Stringer, key resourceType) error {
	w.optionalPath = append(w.optionalPath, optionalEdge{consumer: consumer, key: key})
	_, err := w.resolve(from, key)
	w.optionalPath = w.optionalPath[:len(w.optionalPath)-1] // remove
	return err
}

// resolve schedules whatever has to run before key can be read through from,
// and reports whether the key can be built at all.
//
// from is the view the key is read through, not whoever triggered the walk: the
// same node can be reached from different places, and its own params are
// always read from the scope it belongs to.
//
// A decorator alone does not make a key buildable: it rewrites what a
// constructor produced, so a key with a decorator but no constructor is still
// missing.
func (w *keyWalk) resolve(from *Scope, key resourceType) (bool, error) {
	ctors := from.lookupConstructor(key)
	if len(ctors) == 0 {
		return false, nil
	}

	// A decorator rewrites the value the constructors build, so it is emitted
	// after them; its own params pull those constructors in through the
	// recursion, which is also why the loop below is skipped when one is found.
	d := from.lookupDecorator(key)
	if d != nil && d == w.currentDecorator {
		// The decorator is reading the key it takes over, which is the ordinary
		// case: a decorator normally takes the value it replaces as its own
		// input. Skipping it is not the whole answer -- a decoration above it
		// still has to run, because that decoration is this one's input -- so
		// the search continues from just above the decorator.
		if up := d.scope.parentScope; up != nil {
			d = up.lookupDecorator(key)
		} else {
			d = nil
		}
	}

	if d != nil {
		if w.state[d] == _gray {
			// Still on the path: two decorators need each other's output, so
			// neither of them can go first.
			return true, fmt.Errorf(
				"dig: decorator %v needs %v, but decorator %v takes that over "+
					"and is still running: resolve the cycle",
				w.currentDecorator, key, d)
		}
		return true, w.visitDecorator(d)
	}

	for _, cn := range ctors {
		if err := w.visit(cn); err != nil {
			return true, err
		}
	}
	return true, nil
}

// visit appends cn to the order after visiting all its dependencies.
func (w *keyWalk) visit(cn *constructorNode) error {
	switch w.state[cn] {
	case _black: // already emitted, nothing to do
		return nil
	case _gray: // reached again while still on the path
		return w.cycleError(cn)
	case _white:
		// Fresh node: fall through to the walk below.
	}
	w.state[cn] = _gray
	w.path = append(w.path, cn)
	defer func() { w.path = w.path[:len(w.path)-1] }()

	// The node's params are read from the scope Resource was called on, not from
	// whoever triggered this visit and not from the scope the node happens to be
	// registered in: an exported constructor lives in the root but was written
	// next to its dependencies.
	reqs := cn.ParamList().Requires()
	for _, key := range reqs.required {
		if err := w.resolveRequired(cn.buildScope, key); err != nil {
			return err
		}
	}
	for _, key := range reqs.optional {
		if err := w.resolveOptional(cn.buildScope, cn, key); err != nil {
			return err
		}
	}

	w.state[cn] = _black
	w.order = append(w.order, cn)
	return nil
}

// visitDecorator emits dn after the keys its params need.
func (w *keyWalk) visitDecorator(dn *decoratorNode) error {
	if w.state[dn] == _black { // already emitted, nothing to do
		return nil
	}
	// A gray decorator never gets here: resolve either lets it through because
	// it is the one being visited, or reports it as a cycle.
	w.state[dn] = _gray
	w.path = append(w.path, dn)
	defer func() { w.path = w.path[:len(w.path)-1] }()

	// While this decorator's params resolve, it is the one reading its own key.
	previous := w.currentDecorator
	w.currentDecorator = dn
	defer func() { w.currentDecorator = previous }()

	// A decorator reads its params from the scope it decorates in: that is the
	// scope whose values it rewrites, and the only one whose view of them it can
	// see.
	reqs := dn.paramList.Requires()
	for _, key := range reqs.required {
		if err := w.resolveRequired(dn.scope, key); err != nil {
			return err
		}
	}
	for _, key := range reqs.optional {
		if err := w.resolveOptional(dn.scope, dn, key); err != nil {
			return err
		}
	}

	w.state[dn] = _black
	w.order = append(w.order, dn)
	return nil
}

// addMissing records key once.
func (w *keyWalk) addMissing(key resourceType) {
	if _, ok := w.seen[key]; ok {
		return
	}
	w.seen[key] = struct{}{}
	w.missing = append(w.missing, key)
}

// err reports the missing providers collected so far, if any.
func (w *keyWalk) err() error {
	if len(w.missing) == 0 {
		return nil
	}
	// TODO handle error
	return &dag.MissingProviderError[resourceType]{Resources: w.missing}
}

// cycleError reports a node reached while it is still on the DFS path.
//
// How the path got here decides how this reads. Through an optional edge it is
// a wiring mistake: the user asked for something that cannot be satisfied, and
// naming that edge is the whole fix. Without one the cycle is really there, and
// the only question is why registration did not catch it: it cannot see across
// scopes, and [DeferAcyclicVerification] asks it not to look. Both are things
// the user configured, so the result is a configuration error they can act on,
// reported the same way as the one registration produces.
func (w *keyWalk) cycleError(target runner) error {
	if len(w.optionalPath) == 0 {
		return errCycleDetected{
			Message: "cycle detected",
			Path:    w.cyclePath(target),
		}
	}

	// The innermost edge is the one that closed the loop.
	edge := w.optionalPath[len(w.optionalPath)-1]
	consumer := "the function being set up"
	if edge.consumer != nil {
		consumer = fmt.Sprint(edge.consumer)
	}
	return fmt.Errorf(
		"dig: %v optionally needs %v, but following that edge leads back to %v: "+
			"resolve the cycle", consumer, edge.key, target)
}

// cyclePath names the nodes from where target first went gray around to target
// again. target is on the path by construction: it is still gray, which is what
// sent the walk here.
func (w *keyWalk) cyclePath(target runner) []string {
	start := 0
	for i, n := range w.path {
		if n == target {
			start = i
			break
		}
	}

	names := make([]string, 0, len(w.path)-start+1)
	for _, n := range w.path[start:] {
		names = append(names, fmt.Sprint(n))
	}
	return append(names, fmt.Sprint(target))
}
