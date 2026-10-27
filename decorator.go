package dig

import (
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"
	"github.com/n24-x/dig/internal/dot"
)

type decoratorNode struct {
	decorator  any
	dtype      reflect.Type
	dotID      dot.ConstructorID
	location   *digreflect.Func
	paramList  paramList
	resultList resultList
	scope      *Scope
	called     bool
	observer   Observer
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
