package dig

import (
	"math/rand/v2"
	"reflect"

	dag "github.com/n24-x/dag-go"
)

type Scope struct {
	name string

	dag *dag.DefaultGraph[resourceType]

	values      map[resourceType]reflect.Value
	valueGroups map[resourceType][]reflect.Value

	fnCaller fnCaller

	recoverFromPanics        bool
	deferAcyclicVerification bool

	// TODO
	// parentScope *Scope
}

// fnCaller produces the results of calling fn with args.
//
// The scope holds one instead of calling reflect.Value.Call directly so that
// the container can substitute a different way of producing results; DryRun
// uses it to skip user code entirely.
type fnCaller func(fn reflect.Value, args []reflect.Value) (results []reflect.Value)

var _ func() *Scope = newScope

var _ interface {
	lookupValue(resourceType) (reflect.Value, bool)
	Setup(function any, opts ...Option[*SetupOptions]) error
} = (*Scope)(nil)

var _ valueWriter = (*Scope)(nil)

func newScope() *Scope {
	return &Scope{
		dag:         dag.New[resourceType](),
		values:      make(map[resourceType]reflect.Value),
		valueGroups: make(map[resourceType][]reflect.Value),
		fnCaller:    defaultFnCaller,
	}
}

// lookupValue returns the value built for t, if any.
//
// Multi-scope lookup (walking up parentScope) belongs to the scope-tree work;
// it will live inside this method so the signature does not change.
func (sp *Scope) lookupValue(typ resourceType) (reflect.Value, bool) {
	// TODO walking up parentScope
	v, ok := sp.values[typ]
	return v, ok
}

// lookupGroup returns the values collected for the group key.
//
// The order is deliberately unstable: users must not depend on the order in
// which the group's providers were registered, so the slice is shuffled every
// time it is read. A copy is shuffled rather than the stored slice, otherwise
// repeated reads would fight over the stored order and later submissions would
// append to an already scrambled list.
//
// Multi-scope lookup (merging the ancestors' groups) belongs to the
// scope-tree work; it will live inside this method so the signature does not
// change.
func (sp *Scope) lookupGroup(key resourceType) []reflect.Value {
	// TODO walking up parentScope
	return shuffledCopy(sp.valueGroups[key])
}

// shuffledCopy returns a shuffled copy of items, leaving items untouched.
func shuffledCopy(items []reflect.Value) []reflect.Value {
	out := make([]reflect.Value, len(items))
	for i, j := range rand.Perm(len(items)) {
		out[i] = items[j]
	}
	return out
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
