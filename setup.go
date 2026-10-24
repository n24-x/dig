package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"
)

func (ct *Container) Setup(function any, opts ...Option[*SetupOptions]) error {
	return ct.rootScope.Setup(function, opts...)
}

func (sp *Scope) Setup(function any, opts ...Option[*SetupOptions]) error {
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

	ctorOrder, err := sp.ctorRunOrder(funcPL)
	if err != nil {
		return err
	}

	for _, ctor := range ctorOrder {
		if err := ctor.Run(sp); err != nil { // It may be PANIC
			return err
		}
	}

	args, err := funcPL.Args(sp)
	if err != nil {
		return errArgumentsFailed{Func: digreflect.InspectFunc(function), Reason: err}
	}

	out := sp.callFn(fval, args)

	err = findError(out)
	if err != nil {
		return err
	}

	return nil
}

// ctorRunOrder returns the constructors that must run before function's
// arguments can be assembled, in the order they have to run.
//
// See [ctorWalk] for how the order is computed.
func (sp *Scope) ctorRunOrder(funcPL paramList) ([]*constructorNode, error) {
	w := newCtorWalk(sp)

	reqs := funcPL.Requires()
	for _, key := range reqs.Required {
		providers := sp.dag.Provide(key)
		if len(providers) == 0 {
			w.addMissing(key)
			continue
		}
		for _, id := range providers {
			if err := w.visit(id); err != nil {
				return nil, err
			}
		}
	}
	for _, key := range reqs.Optional {
		for _, id := range sp.dag.Provide(key) { // nobody provides it => no node => skip
			if err := w.visit(id); err != nil {
				return nil, err
			}
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
