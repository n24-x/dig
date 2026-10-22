package dig

import (
	"reflect"

	dag "github.com/n24-x/dag-go"
)

type Scope struct {
	name string

	dag *dag.DefaultGraph[resourceType]

	values      map[resourceType]reflect.Value
	valueGroups map[resourceType][]reflect.Value

	callFn callFn

	// TODO
	// recoverFromPanics        bool
	// deferAcyclicVerification bool
	// parentScope *Scope
}

type callFn func(fn reflect.Value, args []reflect.Value) (results []reflect.Value)

var _ func() *Scope = newScope

var _ interface {
	getValue(resourceType) (reflect.Value, bool)
} = (*Scope)(nil)

var _ valueWriter = (*Scope)(nil)

func newScope() *Scope {
	return &Scope{
		dag:         dag.New[resourceType](),
		values:      make(map[resourceType]reflect.Value),
		valueGroups: make(map[resourceType][]reflect.Value),
		callFn:      defaultCallFn,
	}
}

// getValue returns the value built for t, if any.
//
// Multi-scope lookup (walking up parentScope) belongs to the scope-tree work;
// it will live inside this method so the signature does not change.
func (sp *Scope) getValue(typ resourceType) (reflect.Value, bool) {
	// TODO walking up parentScope
	v, ok := sp.values[typ]
	return v, ok
}

// setValue overwrites the value stored for key.
//
// Overwriting is safe: checkResource rejects duplicate non-group keys at
// registration time, so at most one node ever writes a given key.
func (sp *Scope) setValue(key resourceType, v reflect.Value) {
	sp.values[key] = v
}

// submitGroup appends v to the group stored under key.
func (sp *Scope) submitGroup(key resourceType, v reflect.Value) {
	sp.valueGroups[key] = append(sp.valueGroups[key], v)
}
