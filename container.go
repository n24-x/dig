package dig

// A Container is the root Scope that represents the top-level scope.
type Container struct {
	rootScope *Scope
}

var _ func(opts ...Option[*Container]) *Container = New

func New(opts ...Option[*Container]) *Container {
	ct := &Container{
		rootScope: newScope(),
	}

	for _, opt := range opts {
		opt(ct)
	}

	return ct
}
