package dig

// A Container is the root Scope that represents the top-level scope.
type Container struct {
	rootScope *Scope
}

var (
	_ func(opts ...Option[*Container]) *Container = New

	_ func() Option[*Container] = DryRun
	_ func() Option[*Container] = RecoverFromPanics
	_ func() Option[*Container] = DeferAcyclicVerification
)

func New(opts ...Option[*Container]) *Container {
	ct := &Container{
		rootScope: newScope(),
	}
	ct.rootScope.name = _rootScopeName

	for _, opt := range opts {
		opt(ct)
	}

	return ct
}

// DryRun makes the container resolve dependencies without running any user
// code: constructors and the function passed to Setup both get zero values
// instead of being called. Registration and ordering are unaffected.
func DryRun() Option[*Container] {
	return func(ct *Container) {
		if ct == nil {
			return
		}
		ct.rootScope.fnCaller = dryFnCaller
	}
}

// RecoverFromPanics makes the container turn a panic raised by user code into
// a PanicError instead of letting it escape.
func RecoverFromPanics() Option[*Container] {
	return func(ct *Container) {
		if ct == nil {
			return
		}
		ct.rootScope.recoverFromPanics = true
	}
}

// DeferAcyclicVerification adds resources without checking the graph for cycles
// on every registration. A cycle then surfaces when the graph is first walked
// for an execution order.
func DeferAcyclicVerification() Option[*Container] {
	return func(ct *Container) {
		if ct == nil {
			return
		}
		ct.rootScope.deferAcyclicVerification = true
	}
}
