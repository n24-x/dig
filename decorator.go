package dig

import (
	"fmt"
	"reflect"
	"time"

	"github.com/n24-x/dig/internal/digreflect"
)

type decoratorNode struct {
	decorator  any
	dtype      reflect.Type
	location   *digreflect.Func
	paramList  paramList
	resultList resultList
	scope      *Scope
	called     bool
	observer   Observer
}

var _ interface {
	fmt.Stringer
	runner
} = (*decoratorNode)(nil)

var _ func(any, *Scope) (*decoratorNode, error) = newDecoratorNode

// newDecoratorNode parses dtor into a decorator node bound to sp.
//
// It reads nothing from DecorateOptions: the observer is assigned by the
// caller after registration succeeds, so a rejected decorator never carries
// one. Everything this node needs to run comes from the function itself.
func newDecoratorNode(dtor any, sp *Scope) (*decoratorNode, error) {
	dval := reflect.ValueOf(dtor)
	dtype := dval.Type()

	resultList, err := newResultList(dtype, resultOptions{})
	if err != nil {
		return nil, err
	}

	if len(resultList.Provides()) == 0 {
		return nil, newErrInvalidInput(
			fmt.Sprintf("decorator %v must have at least one non-error type result", dtype), nil)
	}

	paramList, err := newParamList(dtype)
	if err != nil {
		return nil, err
	}

	location := digreflect.InspectFunc(dtor)

	dn := &decoratorNode{
		decorator:  dtor,
		dtype:      dtype,
		location:   location,
		paramList:  paramList,
		resultList: resultList,
		scope:      sp,
	}
	return dn, nil
}

// Run reads the decorator's arguments, calls it, and stores the results as
// decorations. It runs at most once: later calls are no-ops.
//
// Arguments are read from dn.scope, the scope the decorator was registered on,
// not from whichever scope triggered it. A decorator's dependencies resolve
// where it was declared; in a single scope the two are the same.
//
// Results go through staging and are committed only after they were stored
// without error, so a decorator that returns an error leaves the scope
// untouched instead of half decorated.
func (dn *decoratorNode) Run(sp *Scope) (err error) {
	if dn.called {
		return nil
	}

	args, err := dn.paramList.Args(dn.scope)
	if err != nil {
		return errArgumentsFailed{Func: dn.location, Reason: err}
	}

	ob := dn.observer
	obName := dn.location.Dotted()
	if ob.Announce != nil {
		ob.Announce(obName)
	}

	start := time.Now()
	if ob.Report != nil {
		defer func() {
			ob.Report(ObserveInfo{Name: obName, Error: err, ElapsedTime: time.Since(start)})
		}()
	}

	if dn.scope.recoverFromPanics {
		defer func() {
			if p := recover(); p != nil {
				err = PanicError{Func: dn.location, Panic: p}
			}
		}()
	}

	staging := newStagingValues()
	results := dn.scope.fnCaller(reflect.ValueOf(dn.decorator), args)
	if err = dn.resultList.StoreResults(staging, true, results); err != nil {
		return errDecorateFailed{Func: dn.location, Reason: err}
	}

	staging.Commit(dn.scope)
	dn.called = true
	return nil
}

func (dn *decoratorNode) String() string {
	return fmt.Sprintf("[%v]", dn.location)
}

var _ func(*DecoratorReceiptInfo) Option[*DecorateOptions] = DecoratorReceipt

func DecorateObserver(ob Observer) Option[*DecorateOptions] {
	return func(do *DecorateOptions) {
		if do == nil {
			return
		}
		do.observer = ob
	}
}

// DecoratorReceiptInfo is the receipt of a successful decorator registration.
//
// It records what the container understood from the decorator: the resources it
// consumes and the ones it takes over.
type DecoratorReceiptInfo struct {
	Params  []*ParamInfo
	Results []*ResultInfo
}

// DecoratorReceipt writes the receipt of this decorator registration into info.
func DecoratorReceipt(info *DecoratorReceiptInfo) Option[*DecorateOptions] {
	return func(do *DecorateOptions) {
		if do == nil {
			return
		}
		do.info = info
	}
}
