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

	// registry is what the scope was told to build; store is what running it
	// produced. Every traversal reads registry and nothing else, so a walk can
	// see the whole shape of a container before any of it has run.
	registry scopeRegistry
	store    scopeStore

	// cfg is the whole of what a child takes over from its parent, and nothing
	// else: it is fixed when the scope is created and only read afterwards.
	// Keeping it in one struct is what makes the inheritance one assignment, so
	// a new field cannot be inherited only where somebody remembered to add a
	// line in NewChildScope.
	cfg scopeConfig
}

var _ interface {
	Setup(function any, opts ...Option[*SetupOptions]) error
	Validate() error
	NewChildScope(name string) *Scope
	lookupRootScope() *Scope
	lookupValue(resourceType) (reflect.Value, bool)
	lookupGroup(key resourceType) []reflect.Value
	lookupConstructor(key resourceType) []*constructorNode
	hasLocalConstructor(key resourceType) bool
	lookupDecorator(key resourceType) *decoratorNode
} = (*Scope)(nil)

var _ valueWriter = (*Scope)(nil)

var _ func() *Scope = newScope

// scopeStore is what this scope built: the values its constructors returned and
// the ones its decorators rewrote, each single value and each group.
//
// A child scope starts with an empty store and never reads its parent's: a
// value belongs to the scope whose constructor produced it, and a lookup walks
// up parentScope to find it rather than merging the tables.
type scopeStore struct {
	values          map[resourceType]reflect.Value
	decoratedValues map[resourceType]reflect.Value

	valueGroups     map[resourceType][]reflect.Value
	decoratedGroups map[resourceType]reflect.Value
}

// scopeRegistry is the registration state of one scope: the constructors it
// holds, the decorations it applies, and whether the constructors still owe a
// cycle check.
//
// Nothing here is produced by running user code, which is what separates it
// from the store: a scope that has never been set up has already registered
// everything it will ever register.
type scopeRegistry struct {
	// ctors is the graph the constructors are nodes of: Add is what reports a
	// cycle, and Provide is what a lookup asks for the nodes behind a key.
	ctors *dag.DefaultGraph[resourceType]
	dtors map[resourceType]*decoratorNode

	// needsCheckDependencyCycle reports that a constructor went in unchecked
	// since the last whole-graph scan. Registration sets it and Setup clears it,
	// so it belongs to one scope rather than to scopeConfig.
	needsCheckDependencyCycle bool
}

// scopeConfig is how user code is called and how its failures are treated.
//
// Everything here is inherited by a child scope and never changes after the
// container is built. A field that a single scope sets and then clears is state
// rather than configuration, and belongs on scopeRegistry instead.
type scopeConfig struct {
	fnCaller                  fnCaller
	recoverFromPanics         bool
	deferCheckDependencyCycle bool
}

// fnCaller produces the results of calling fn with args.
//
// The scope holds one instead of calling reflect.Value.Call directly so that
// the container can substitute a different way of producing results; DryRun
// uses it to skip user code entirely.
type fnCaller func(fn reflect.Value, args []reflect.Value) (results []reflect.Value)

func newScope() *Scope {
	return &Scope{
		registry: scopeRegistry{
			ctors: dag.New[resourceType](),
			dtors: make(map[resourceType]*decoratorNode),
		},
		store: scopeStore{
			values:          make(map[resourceType]reflect.Value),
			decoratedValues: make(map[resourceType]reflect.Value),
			valueGroups:     make(map[resourceType][]reflect.Value),
			decoratedGroups: make(map[resourceType]reflect.Value),
		},
		cfg: scopeConfig{fnCaller: defaultFnCaller},
	}
}

// Validate reports the dependencies this scope cannot satisfy.
//
// Every constructor registered here is walked, not only the ones a Setup would
// reach, so a constructor that nothing asks for yet still has its required
// dependencies checked. Nothing is run: Validate reads the registry and stops
// there, so it costs a walk and no user code.
//
// A key is answered the way a reader at this scope would answer it, so a
// dependency an ancestor provides is satisfied and one a child provides is not.
// Each constructor's own params are read through its build scope, which is the
// scope Resource was called on: an exported node is checked where it was
// written, not where it landed.
//
// Decorations are outside the check. A decorator takes parameters, but they are
// resolved only when something reads the key it takes over, and the walk here
// never does that: it answers every key with the constructors behind it. So a
// decorator whose own parameters cannot be satisfied passes Validate and fails
// at the first read instead.
func (sp *Scope) Validate() error {
	w := newConstructorWalk(sp.registry.ctors.Count())
	for _, cn := range sp.localConstructors() {
		if err := w.visit(cn); err != nil {
			return err
		}
	}
	return w.err()
}

// NewChildScope returns a new scope whose parent is sp.
//
// The child reads everything sp provides and decorates; sp never reads anything
// the child registers, and neither do sp's other children. sp itself is not
// modified by this call, so building a tree costs one allocation per node.
//
// Everything in scopeConfig is inherited. Both the registry and the store start
// empty -- no constructors, no decorations, no values -- and that emptiness is
// what makes a scope private by default; visibility is decided later, when a
// lookup walks up parentScope, not here.
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
	child.cfg = sp.cfg
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

// lookupConstructor returns the constructors that provide key, if any, walking
// up parentScope.
//
// The two kinds of key aggregate differently, because they mean different
// things. A single value is one answer, so the nearest scope that has one
// wins and the rest are not consulted: a scope that provides a type its parent
// also provides hides the parent's provider instead of competing with it, and
// the parent keeps reading its own. Stopping at the first hit is what makes
// that shadow work; collecting both would leave two constructors for one key
// with nothing to choose between them.
//
// A group is a collection, so every scope on the way up contributes its own
// members. Nearest first, so the order a reader sees is stable before the
// shuffle that [Scope.lookupGroup] applies.
func (sp *Scope) lookupConstructor(key resourceType) []*constructorNode {
	if key.groupName != "" {
		var nodes []*constructorNode
		for s := sp; s != nil; s = s.parentScope {
			nodes = append(nodes, s.lookupLocalConstructors(key)...)
		}
		return nodes
	}

	for s := sp; s != nil; s = s.parentScope {
		if nodes := s.lookupLocalConstructors(key); len(nodes) > 0 {
			return nodes
		}
	}
	return nil
}

// lookupLocalConstructors returns the constructors this scope itself holds for
// key, ignoring the ancestors.
func (sp *Scope) lookupLocalConstructors(key resourceType) []*constructorNode {
	ids := sp.registry.ctors.Provide(key)
	nodes := make([]*constructorNode, 0, len(ids))
	for _, id := range ids {
		nodes = append(nodes, sp.registry.ctors.At(id).(*constructorNode))
	}
	return nodes
}

// lookupDecorator returns the decorator that applies to key, if any.
func (sp *Scope) lookupDecorator(key resourceType) *decoratorNode {
	return sp.registry.dtors[key]
}

// lookupValue returns the value for key, if any, walking up parentScope.
//
// The nearest scope holding anything for key wins: the walk stops at the first
// scope with a value, so a scope that provides a type its parent also provides
// reads its own and the parent keeps reading its own.
//
// Within that scope a decoration is read before a plain value. A decoration
// rewrites what that scope's constructor built, so once the decorator has run
// its output is what a reader should see; the plain value stays behind it, so a
// scope that registered a decorator but has not run it yet still finds what the
// constructor stored.
//
// One pass, not two. A decoration belongs to the scope that owns the provider --
// [Scope.lookupDecorator] reads only this scope, and registering one without a
// constructor here is rejected -- so an ancestor's decoration can never outrank
// a nearer scope's own value. Asking every scope for a decoration before asking
// any for a plain value would let a parent's decoration reach across a child's
// provider and hide a value the child built itself.
//
// Shadowing and decoration agree, then: a child that provides a key its parent
// decorates reads its own undecorated value, because the parent's decoration
// rewrites the parent's value and there is no path from the child's provider
// through it.
func (sp *Scope) lookupValue(typ resourceType) (reflect.Value, bool) {
	for s := sp; s != nil; s = s.parentScope {
		if v, ok := s.store.decoratedValues[typ]; ok {
			return v, true
		}
		if v, ok := s.store.values[typ]; ok {
			return v, true
		}
	}
	return reflect.Value{}, false
}

// lookupGroup returns the members collected for the group key, walking up
// parentScope.
//
// A group is assembled from every scope on the chain, so a decoration of it is
// read only at the scope that declared it. That decoration is the whole group
// as its own scope sees it; handing it down would leave the members a nearer
// scope contributed with nowhere to go. A child therefore collects the raw
// members, from itself upwards, even when an ancestor decorated the group.
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
func (sp *Scope) lookupGroup(key resourceType) []reflect.Value {
	// spread turns a stored decorated group back into individual members.
	// Rebuilding them one by one also normalises a named slice type: a
	// decorator may return type Handlers []string where the consumer asked
	// for []string.
	spread := func(whole reflect.Value) []reflect.Value {
		items := make([]reflect.Value, whole.Len())
		for i := range whole.Len() {
			items[i] = whole.Index(i)
		}
		return items
	}

	// shuffledCopy returns a shuffled copy of items, leaving items untouched.
	shuffledCopy := func(items []reflect.Value) []reflect.Value {
		out := make([]reflect.Value, len(items))
		for i, j := range rand.Perm(len(items)) {
			out[i] = items[j]
		}
		return out
	}

	if whole, ok := sp.store.decoratedGroups[key]; ok {
		return spread(whole)
	}

	var members []reflect.Value
	for s := sp; s != nil; s = s.parentScope {
		members = append(members, s.store.valueGroups[key]...)
	}
	return shuffledCopy(members)
}

// localConstructors returns every constructor this scope registered, in
// registration order.
//
// The graph hands out node IDs rather than the nodes, so walking all of them
// means walking the ID range it assigned. [Scope.lookupLocalConstructors] does
// not need this: it is asked for one key, and the graph answers with that key's
// IDs.
func (sp *Scope) localConstructors() []*constructorNode {
	count := sp.registry.ctors.Count()
	nodes := make([]*constructorNode, 0, count)
	for id := 1; id <= count; id++ {
		nodes = append(nodes, sp.registry.ctors.At(dag.NodeID(id)).(*constructorNode))
	}
	return nodes
}

// hasLocalConstructor reports whether this scope itself provides key.
//
// It stops at this scope on purpose. A child scope providing a type its parent
// already provides is a legal shadow: the child reads its own value and the
// parent keeps its own, so a registration guard must not look at the ancestor
// chain. [Scope.lookupConstructor] does look at it, and is the wrong call here.
func (sp *Scope) hasLocalConstructor(key resourceType) bool {
	return len(sp.registry.ctors.Provide(key)) > 0
}

// setValue overwrites the value stored for key.
//
// Overwriting is safe: checkResource rejects duplicate non-group keys at
// registration time, so at most one node ever writes a given key.
func (sp *Scope) setValue(key resourceType, v reflect.Value) {
	sp.store.values[key] = v
}

// setDecoratedValue records what a decorator produced for key.
func (sp *Scope) setDecoratedValue(key resourceType, v reflect.Value) {
	sp.store.decoratedValues[key] = v
}

// submitGroupValue adds one member to the group stored under key.
func (sp *Scope) submitGroupValue(key resourceType, v reflect.Value) {
	sp.store.valueGroups[key] = append(sp.store.valueGroups[key], v)
}

// submitDecoratedGroup replaces the whole group stored under key with slice.
func (sp *Scope) submitDecoratedGroup(key resourceType, slice reflect.Value) {
	sp.store.decoratedGroups[key] = slice
}
