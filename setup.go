package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"

	"github.com/n24-x/dag-go"
)

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

	var ctorOrder []dag.NodeID
	seen := make(map[dag.NodeID]bool) // TODO go1.28 change to container/set
	for _, key := range funcPL.Requires() {
		ids, err := dag.Resolve(sp.dag, key)
		if err != nil {
			return err // TODO handle *dag.MissingProviderError / CycleError
		}
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				ctorOrder = append(ctorOrder, id)
			}
		}
	}

	for _, id := range ctorOrder {
		ctor := sp.dag.At(id).(*constructorNode)
		if err := ctor.Call(sp); err != nil { // It may be PANIC
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
