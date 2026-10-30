package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"
)

func (ct *Container) Setup(function any, opts ...Option[*SetupOptions]) error {
	return ct.rootScope.Setup(function, opts...)
}

// Setup runs function after building its dependencies: every constructor the
// function's params reach is executed first, in dependency order, and the
// resulting values are read out of the scope to form the call's arguments.
func (sp *Scope) Setup(function any, opts ...Option[*SetupOptions]) (err error) {
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

	// DeferAcyclicVerification skips the check at registration time, so the
	// graph may still hold a cycle. Verify before anything walks it: keyWalk
	// would otherwise report the cycle as an internal error, and the point of
	// deferring is that the user gets the same errCycleDetected either way.
	if sp.needsAcyclicCheck {
		if err = sp.dag.VerifyAcyclic(); err != nil {
			return errCycleDetected{
				Message: "cycle detected",
				Cause:   err,
			}
		}
		sp.needsAcyclicCheck = false
	}

	order, err := sp.runOrder(funcPL)
	if err != nil {
		return err
	}

	for _, node := range order {
		if err = node.Run(); err != nil {
			return err
		}
	}

	args, err := funcPL.Args(sp)
	if err != nil {
		return errArgumentsFailed{Func: digreflect.InspectFunc(function), Reason: err}
	}

	// Registered after the argument assembly so it covers only the user's own
	// function: everything that could fail before this point reports an error
	// instead of panicking.
	if sp.recoverFromPanics {
		defer func() {
			if p := recover(); p != nil {
				err = PanicError{Func: digreflect.InspectFunc(function), Panic: p}
			}
		}()
	}

	out := sp.fnCaller(fval, args)

	err = findError(out)
	if err != nil {
		return err
	}

	return nil
}

// runner is a node that can run as part of a Setup. Both node kinds know where
// they run from their own fields, so no scope is passed in: whoever triggered
// a node has nothing to say about its view.
type runner interface{ Run() error }

// runOrder returns the nodes that must run before function's arguments can be
// assembled, in the order they have to run. A returned node is a constructor
// or a decorator; both are run the same way.
//
// See [keyWalk] for how the order is computed.
func (sp *Scope) runOrder(funcPL paramList) ([]runner, error) {
	w := newKeyWalk(sp)

	reqs := funcPL.Requires()
	for _, key := range reqs.required {
		if err := w.resolveRequired(key); err != nil {
			return nil, err
		}
	}
	for _, key := range reqs.optional {
		if err := w.resolveOptional(nil, key); err != nil { // nil: the function being set up
			return nil, err
		}
	}

	if err := w.err(); err != nil {
		return nil, err
	}
	return w.order, nil
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
