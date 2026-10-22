package dig

import "reflect"

// A Container is the root Scope that represents the top-level scope.
type Container struct {
	rootScope *Scope
}

var _ func(opts ...Option[*Container]) *Container = New

// setupFn exists ONLY to switch how the container calls user-supplied functions.
type setupFn func(fn reflect.Value, args []reflect.Value) (results []reflect.Value)

func New(opts ...Option[*Container]) *Container {
	ct := &Container{
		rootScope: newScope(),
	}

	for _, opt := range opts {
		opt(ct)
	}

	return ct
}
