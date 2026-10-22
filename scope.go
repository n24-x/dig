package dig

import (
	"reflect"

	dag "github.com/n24-x/dag-go"
)

type Scope struct {
	name string

	dag *dag.DefaultGraph[resourceType]

	values map[resourceType]reflect.Value

	// TODO
	// valueGroups map[resourceType][]reflect.Value
	// recoverFromPanics        bool
	// deferAcyclicVerification bool
	// setupFn
	// parentScope *Scope
}

var _ func() *Scope = newScope

var _ interface {
	getValue(resourceType) (reflect.Value, bool)
} = (*Scope)(nil)

func newScope() *Scope {
	return &Scope{
		dag:    dag.New[resourceType](),
		values: make(map[resourceType]reflect.Value),
		// valueGroups: make(map[resourceType][]reflect.Value),
	}
}

// getValue returns the value built for t, if any.
//
// Multi-scope lookup (walking up parentScope) belongs to the scope-tree work;
// it will live inside this method so the signature does not change.
func (sp *Scope) getValue(typ resourceType) (reflect.Value, bool) {
	v, ok := sp.values[typ]
	return v, ok
}
