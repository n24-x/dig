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

	ctorOrder, err := sp.ctorRunOrder(funcPL)
	if err != nil {
		return err
	}

	for _, id := range ctorOrder {
		ctor := sp.dag.At(id).(*constructorNode)
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
// Required params go first: dag.Resolve returns their whole build path in
// post-order, so every dependency precedes its consumer. Optional params are
// added afterwards, and only when somebody provides them; a param whose key
// nobody provides keeps its zero value and contributes no node.
func (sp *Scope) ctorRunOrder(funcPL paramList) ([]dag.NodeID, error) {
	var order []dag.NodeID
	seen := make(map[dag.NodeID]bool) // TODO go1.28 change to container/set

	reqs := funcPL.Requires()

	for _, key := range reqs.Required {
		ids, err := dag.Resolve(sp.dag, key)
		if err != nil {
			return nil, err // TODO handle *dag.MissingProviderError / CycleError
		}
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				order = append(order, id)
			}
		}
	}

	for _, key := range reqs.Optional {
		if len(sp.dag.Provide(key)) == 0 {
			continue
		}
		ids, err := dag.Resolve(sp.dag, key)
		if err != nil {
			// TODO
			// handle CycleError
			return nil, err
		}
		for _, id := range ids {
			if !seen[id] {
				seen[id] = true
				order = append(order, id)
			}
		}
	}

	return order, nil
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
