package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"
)

// ResourceReceiptInfo is the receipt of a successful resource registration.
//
// It records what the container understood from the constructor: the DAG node
// it became, and the resources it consumes and provides.
type ResourceReceiptInfo struct {
	DagID   DagID
	Params  []*ParamInfo
	Results []*ResultInfo
}

type ParamInfo struct {
	valueType reflect.Type

	valueName string
	optional  bool

	groupName string
}

type ResultInfo struct {
	valueType reflect.Type

	valueName string

	groupName string
}

// String returns a string representation of ParamInfo from a validated constructor.
//
// For example `*sql.DB[name="ro"]` or `[]*sql.DB[group="replicas"]`.
func (p *ParamInfo) String() string {
	typ := p.valueType.String()

	switch {
	case p.groupName != "":
		return fmt.Sprintf("%s[group=%q]", typ, p.groupName)
	case p.optional && p.valueName != "":
		return fmt.Sprintf("%s[optional, name=%q]", typ, p.valueName)
	case p.optional:
		return fmt.Sprintf("%s[optional]", typ)
	case p.valueName != "":
		return fmt.Sprintf("%s[name=%q]", typ, p.valueName)
	default:
		return typ
	}
}

// String returns a string representation of ResultInfo from a validated constructor.
//
// For example `*sql.DB[name="ro"]`.
func (r *ResultInfo) String() string {
	switch {
	case r.groupName != "":
		return fmt.Sprintf("%s[group=%q]", r.valueType, r.groupName)
	case r.valueName != "":
		return fmt.Sprintf("%s[name=%q]", r.valueType, r.valueName)
	default:
		return r.valueType.String()
	}
}

func (ct *Container) Resource(ctor any, opts ...Option[*resourceOptions]) error {
	return ct.rootScope.Resource(ctor, opts...)
}

func (sp *Scope) Resource(ctor any, opts ...Option[*resourceOptions]) error {
	ctype := reflect.TypeOf(ctor)
	if ctype == nil {
		return newErrInvalidInput("dig: nil cannot provide resource", nil)
	}

	if ctype.Kind() != reflect.Func {
		return newErrInvalidInput(
			fmt.Sprintf("must be function, got %v (type %v)",
				ctor, ctype), nil)
	}

	var options resourceOptions
	for _, fn := range opts {
		fn(&options)
	}
	if err := options.Validate(); err != nil {
		return err
	}

	if err := sp.resource(ctor, options); err != nil {
		var badFunc *digreflect.Func
		if options.Location == nil {
			badFunc = digreflect.InspectFunc(ctor)
		} else {
			badFunc = options.Location
		}

		return errResourceAddFailed{
			Func:   badFunc,
			Reason: err,
		}
	}
	return nil
}

func (sp *Scope) resource(ctor any, options resourceOptions) (err error) {
	node, err := newConstructorNode(ctor, constructorOptions{
		Result: resultOptions{
			ValueName: options.ValueName,
			GroupName: options.GroupName,
			As:        options.As,
		},
		Location: options.Location,
	})
	if err != nil {
		return err
	}

	// TODO register node in the DAG, detect duplicate providers, then fill
	// options.Info with the receipt.
	_ = node
	return nil
}

// Name sets the name of the provided value.
//
// TODO cleanup
// Validation is performed once by [Validate] rather than here.
func Name(name string) Option[*resourceOptions] {
	return func(ro *resourceOptions) {
		if ro == nil {
			return
		}
		ro.ValueName = name
	}
}

// Group places the provided value in the given group.
//
// TODO cleanup
// Validation is performed once by [Validate] rather than here.
func Group(groupName string) Option[*resourceOptions] {
	return func(ro *resourceOptions) {
		if ro == nil {
			return
		}
		ro.GroupName = groupName
	}
}

// LocationForPC records the call site that constructed this option. It is used
// to attribute errors to user code instead of to this package.
func LocationForPC(pc uintptr) Option[*resourceOptions] {
	return func(ro *resourceOptions) {
		if ro == nil {
			return
		}
		ro.Location = digreflect.InspectFuncPC(pc)
	}
}

// As makes the produced value available under the given interfaces as well as
// its own type. Each argument must be a pointer to an interface that the value
// implements; because the argument is untyped, this cannot be checked by the
// compiler and is validated by [Validate] and at parse time instead.
func As(types ...any) Option[*resourceOptions] {
	return func(ro *resourceOptions) {
		if ro == nil {
			return
		}
		ro.As = append(ro.As, types...)
	}
}

// ResourceReceipt writes the receipt of this resource registration into info.
//
// The receipt is filled only after the resource has been registered
// successfully. Passing a nil info is a no-op.
func ResourceReceipt(info *ResourceReceiptInfo) Option[*resourceOptions] {
	return func(ro *resourceOptions) {
		if ro == nil {
			return
		}
		ro.Info = info
	}
}
