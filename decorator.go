package dig

import (
	"fmt"
	"reflect"

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

var _ fmt.Stringer = (*decoratorNode)(nil)

var _ func(any, ConstructorOptions) (*constructorNode, error) = newConstructorNode

// newDecoratorNode parses decorator into a decorator node bound to sp.
func newDecoratorNode(dtor any, sp *Scope, opts DecorateOptions) (*decoratorNode, error) {
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

	// TODO move to Decorate
	if opts.info != nil {
		opts.info.Params = paramInfos(paramList.DotParams())
		opts.info.Results = resultInfos(resultList.DotResults())
	}

	dn := &decoratorNode{
		decorator:  dtor,
		dtype:      dtype,
		location:   location,
		paramList:  paramList,
		resultList: resultList,
		scope:      sp,
		observer:   opts.observer,
	}
	return dn, nil
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
