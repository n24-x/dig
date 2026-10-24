package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"

	"github.com/n24-x/dag-go"
)

func (sp *Scope) Setup(function any, opts ...Option[*SetupOptions]) error {
	var options SetupOptions
	for _, fn := range opts {
		fn(&options)
	}

	ftype := reflect.TypeOf(function)
	if ftype == nil {
		return newErrInvalidInput("cannot setup nil function", nil)
	}
	if ftype.Kind() != reflect.Func {
		return newErrInvalidInput(
			fmt.Sprintf("cannot setup non-function %v (type %v)", function, ftype), nil)
	}
	fval := reflect.ValueOf(function)

	funcPL, err := newParamList(ftype)
	if err != nil {
		return err
	}

	if info := options.Info; info != nil {
		params := paramInfos(funcPL.DotParams())
		info.Params = params
	}

	ctorOrder, err := sp.ctorRunOrder(funcPL)
	if err != nil {
		return err
	}

	for _, id := range ctorOrder {
		ctor := sp.dag.At(id).(*constructorNode)
		if err := ctor.Run(sp); err != nil { // It may be PANIC
			return err
		}
	}

	args, err := funcPL.Args(sp)
	if err != nil {
		return errArgumentsFailed{Func: digreflect.InspectFunc(function), Reason: err}
	}

	out := sp.callFn(fval, args)

	err = findError(out)
	if err != nil {
		return err
	}

	return nil
}

// ctorRunOrder returns the constructors that must run before function's
// arguments can be assembled, in the order they have to run.
//
// The graph is walked with a three-colour depth-first search starting from the
// function's own params. Required edges come from the dag; optional edges are
// followed too, but only when somebody provides the key, so an optional dep
// whose key nobody provides contributes no node and the param keeps its zero
// value.
//
// An optional edge is followed only when its provider can still run before the
// consumer. If the provider's required closure leads back to the consumer, the
// two intents contradict each other and the graph is rejected: silently
// handing the consumer a zero value would hide a real mistake in the wiring.
//
// Required edges have no such escape hatch, but dag.Add has already rejected
// cycles over them, so a back edge there is an invariant violation.
func (sp *Scope) ctorRunOrder(funcPL paramList) ([]dag.NodeID, error) {
	const (
		unvisited = iota // zero value of the state slice
		visiting         // on the current DFS path
		done             // already emitted
	)

	state := make([]uint8, sp.dag.Count()+1) // indexed by NodeID; 0 is unused
	order := make([]dag.NodeID, 0, sp.dag.Count())

	var (
		missing    []resourceType
		seenMiss   = make(map[resourceType]struct{})
		recordMiss = func(key resourceType) {
			if _, ok := seenMiss[key]; ok {
				return
			}
			seenMiss[key] = struct{}{}
			missing = append(missing, key)
		}
	)

	var visit func(id dag.NodeID) error
	visit = func(id dag.NodeID) error {
		switch state[id] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("dig: cycle detected while ordering constructors at node %d", id)
		}
		state[id] = visiting

		cn, ok := sp.dag.At(id).(*constructorNode)
		if !ok {
			return fmt.Errorf("dig: node %d is not a constructor", id)
		}

		// Required deps: dag edges. A required dep that nobody provides has
		// no edge and would otherwise be skipped in silence, so check it here.
		for _, key := range cn.Requires() {
			if len(sp.dag.Provide(key)) == 0 {
				recordMiss(key)
			}
		}
		for _, dep := range sp.dag.OutNeighbors(id) {
			if err := visit(dep); err != nil {
				return err
			}
		}

		// Optional deps: no edge in the dag, so resolve the providers by hand.
		// If the provider would need this node to have run first, the two
		// cannot both be satisfied; report it rather than silently handing the
		// consumer a zero value.
		for _, key := range cn.ParamList().Requires().Optional {
			for _, dep := range sp.dag.Provide(key) {
				if sp.requiredReaches(dep, id) {
					return fmt.Errorf(
						"dig: constructor %v optionally needs %v, but its provider %v requires this constructor: resolve the cycle",
						cn, key, sp.dag.At(dep))
				}
				if err := visit(dep); err != nil {
					return err
				}
			}
		}

		state[id] = done
		order = append(order, id)
		return nil
	}

	reqs := funcPL.Requires()
	for _, key := range reqs.Required {
		providers := sp.dag.Provide(key)
		if len(providers) == 0 {
			recordMiss(key)
			continue
		}
		for _, id := range providers {
			if err := visit(id); err != nil {
				return nil, err
			}
		}
	}
	for _, key := range reqs.Optional {
		for _, id := range sp.dag.Provide(key) { // nobody provides it => no node
			if err := visit(id); err != nil {
				return nil, err
			}
		}
	}

	if len(missing) > 0 {
		return nil, &dag.MissingProviderError[resourceType]{Resources: missing}
	}
	return order, nil
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

var _ func(*SetupInfo) Option[*SetupOptions] = SetupReceipt

// SetupReceipt writes [Setup] call func info.
//
// Unlike ResourceReceipt, the receipt is filled as soon as the function has
// been parsed. Passing a nil info is a no-op.
func SetupReceipt(info *SetupInfo) Option[*SetupOptions] {
	return func(so *SetupOptions) {
		if so == nil {
			return
		}
		so.Info = info
	}
}

type SetupOptions struct {
	Info *SetupInfo
}

type SetupInfo struct {
	Params []*ParamInfo
}
