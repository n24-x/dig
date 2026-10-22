package dig

import (
	dag "github.com/n24-x/dag-go"
)

type Scope struct {
	name string

	dag *dag.DefaultGraph[resourceType]

	// TODO
	// values      map[resourceType]reflect.Value
	// valueGroups map[resourceType][]reflect.Value
	// recoverFromPanics        bool
	// deferAcyclicVerification bool
	// setupFn
	// parentScope *Scope
}

var _ func() *Scope = newScope

func newScope() *Scope {
	return &Scope{
		dag: dag.New[resourceType](),
		// values:      make(map[resourceType]reflect.Value),
		// valueGroups: make(map[resourceType][]reflect.Value),
	}
}
