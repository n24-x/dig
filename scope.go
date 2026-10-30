package dig

import (
	"fmt"
	"math/rand/v2"
	"reflect"

	dag "github.com/n24-x/dag-go"
)

const _rootScopeName = "root"

type Scope struct {
	name        string
	parentScope *Scope

	dag        *dag.DefaultGraph[resourceType] // store ctor
	decorators map[resourceType]*decoratorNode // store decorator

	values          map[resourceType]reflect.Value
	decoratedValues map[resourceType]reflect.Value

	valueGroups     map[resourceType][]reflect.Value
	decoratedGroups map[resourceType]reflect.Value

	fnCaller fnCaller

	recoverFromPanics        bool
	deferAcyclicVerification bool
	needsAcyclicCheck        bool
}

// fnCaller produces the results of calling fn with args.
//
// The scope holds one instead of calling reflect.Value.Call directly so that
// the container can substitute a different way of producing results; DryRun
// uses it to skip user code entirely.
type fnCaller func(fn reflect.Value, args []reflect.Value) (results []reflect.Value)

var _ func() *Scope = newScope

var _ interface {
	Setup(function any, opts ...Option[*SetupOptions]) error
	NewChildScope(name string) *Scope
	lookupRootScope() *Scope
	lookupValue(resourceType) (reflect.Value, bool)
	lookupGroup(key resourceType) []reflect.Value
	lookupConstructor(key resourceType) []*constructorNode
	hasOwnConstructor(key resourceType) bool
	lookupDecorator(key resourceType) *decoratorNode
} = (*Scope)(nil)

var _ valueWriter = (*Scope)(nil)

func newScope() *Scope {
	return &Scope{
		dag:             dag.New[resourceType](),
		decorators:      make(map[resourceType]*decoratorNode),
		values:          make(map[resourceType]reflect.Value),
		valueGroups:     make(map[resourceType][]reflect.Value),
		decoratedValues: make(map[resourceType]reflect.Value),
		decoratedGroups: make(map[resourceType]reflect.Value),
		fnCaller:        defaultFnCaller,
	}
}

// NewChildScope returns a new scope whose parent is sp.
//
// The child reads everything sp provides and decorates; sp never reads anything
// the child registers, and neither do sp's other children. sp itself is not
// modified by this call, so building a tree costs one allocation per node.
//
// Only the way user code is called is inherited -- the fnCaller, the panic
// recovery and the deferred cycle check. Everything a scope stores starts
// empty: the graph, the decorators, the values and the groups. That emptiness
// is what makes a scope private by default; visibility is decided later, when
// a lookup walks up parentScope, not here.
//
// NewChildScope panics when name is empty or is the root scope's name.
// Diagnostics print scope names, so a nameless scope would carry no label at
// all, and a second scope answering to "root" would make that label ambiguous.
func (sp *Scope) NewChildScope(name string) *Scope {
	switch {
	case name == "":
		panic("dig: NewChildScope: the name must not be empty, it labels the " +
			"scope in diagnostics")
	case name == _rootScopeName:
		panic(fmt.Sprintf(
			"dig: NewChildScope(%q): that name is reserved for the root scope", name))
	}

	child := newScope()
	child.name = name
	child.parentScope = sp
	child.fnCaller = sp.fnCaller
	child.recoverFromPanics = sp.recoverFromPanics
	child.deferAcyclicVerification = sp.deferAcyclicVerification
	return child
}

// lookupRootScope returns the top of the tree sp belongs to: the scope whose
// parentScope is nil, and the one an exported resource is registered into.
func (sp *Scope) lookupRootScope() *Scope {
	root := sp
	parent := sp.parentScope
	for parent != nil {
		root = parent
		parent = parent.parentScope
	}
	return root
}

// lookupConstructor returns the constructors that provide key, if any.
//
// Multi-scope lookup (walking up parentScope) belongs to the scope-tree work;
// it will live inside this method so the signature does not change.
func (sp *Scope) lookupConstructor(key resourceType) []*constructorNode {
	// TODO walking up parentScope
	ids := sp.dag.Provide(key)
	nodes := make([]*constructorNode, 0, len(ids))
	for _, id := range ids {
		nodes = append(nodes, sp.dag.At(id).(*constructorNode))
	}
	return nodes
}

// hasOwnConstructor reports whether this scope itself provides key.
//
// It stops at this scope on purpose. A child scope providing a type its parent
// already provides is a legal shadow: the child reads its own value and the
// parent keeps its own, so a registration guard must not look at the ancestor
// chain. [Scope.lookupConstructor] does look at it, and is the wrong call here.
func (sp *Scope) hasOwnConstructor(key resourceType) bool {
	return len(sp.dag.Provide(key)) > 0
}

// lookupDecorator returns the decorator that supplies key, if any.
//
// Multi-scope lookup (walking up parentScope) belongs to the scope-tree work;
// it will live inside this method so the signature does not change.
func (sp *Scope) lookupDecorator(key resourceType) *decoratorNode {
	// TODO walking up parentScope
	return sp.decorators[key]
}

// lookupValue returns the value for key, if any.
//
// A decoration shadows the provider's value rather than replacing it: the
// decorated table is consulted first and the provider's value stays behind it,
// so a scope that registered a decorator but has not run it yet still reads
// what the constructor built.
//
// Multi-scope lookup (walking up parentScope) belongs to the scope-tree work;
// it will live inside this method so the signature does not change.
func (sp *Scope) lookupValue(typ resourceType) (reflect.Value, bool) {
	// TODO walking up parentScope
	if v, ok := sp.decoratedValues[typ]; ok {
		return v, true
	}

	v, ok := sp.values[typ]
	return v, ok
}

// lookupGroup returns the members collected for the group key.
//
// A decorated group comes back in the order the decorator returned, and it is
// the whole group rather than a part of it. The shuffle below exists so that
// nobody depends on the order providers happened to be registered in; a
// decorator's order is a decision, so it is kept as made.
//
// Otherwise the order is deliberately unstable: the slice is shuffled every
// time it is read. A copy is shuffled rather than the stored slice, otherwise
// repeated reads would fight over the stored order and later submissions would
// append to an already scrambled list.
//
// Multi-scope lookup (merging the ancestors' groups) belongs to the
// scope-tree work; it will live inside this method so the signature does not
// change.
func (sp *Scope) lookupGroup(key resourceType) []reflect.Value {
	// TODO walking up parentScope
	if whole, ok := sp.decoratedGroups[key]; ok {
		// Spread the stored slice back into members. Rebuilding it from the
		// element type also normalises a named slice type: a decorator may
		// return type Handlers []string where the consumer asked for []string.
		items := make([]reflect.Value, whole.Len())
		for i := range whole.Len() {
			items[i] = whole.Index(i)
		}
		return items
	}

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

// setDecoratedValue records what a decorator produced for key.
func (sp *Scope) setDecoratedValue(key resourceType, v reflect.Value) {
	sp.decoratedValues[key] = v
}

// submitGroupValue adds one member to the group stored under key.
func (sp *Scope) submitGroupValue(key resourceType, v reflect.Value) {
	sp.valueGroups[key] = append(sp.valueGroups[key], v)
}

// submitDecoratedGroup replaces the whole group stored under key with slice.
func (sp *Scope) submitDecoratedGroup(key resourceType, slice reflect.Value) {
	sp.decoratedGroups[key] = slice
}
