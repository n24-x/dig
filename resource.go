package dig

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"
	"github.com/n24-x/dig/internal/dot"

	dag "github.com/n24-x/dag-go"
)

var (
	_ func(*Container, any, ...Option[*resourceOptions]) error = (*Container).Resource
	_ func(*Scope, any, ...Option[*resourceOptions]) error     = (*Scope).Resource
	_ func(*Scope, any, resourceOptions) error                 = (*Scope).resource
	_ func(*Scope, []resourceType) string                      = (*Scope).checkResource
)

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

func (sp *Scope) resource(ctor any, options resourceOptions) error {
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

	if reason := sp.checkResource(node.Provides()); reason != "" {
		return newErrInvalidInput(reason, nil)
	}

	id, err := sp.registerNode(node)
	if err != nil {
		return err
	}

	if info := options.Info; info != nil {
		info.DagID = DagID(id)
		info.Params = paramInfos(node.ParamList().DotParams())
		info.Results = resultInfos(node.ResultList().DotResults())
	}

	return nil
}

// registerNode adds node to the scope's graph and returns its ID.
//
// With DeferAcyclicVerification the node goes in unchecked and the scope
// remembers that the graph still owes a whole-graph scan, which Setup pays for
// before it runs anything. Without it, Add itself reports the cycle.
//
// Both paths reject a cycle with the same error type, so callers cannot tell
// which one caught it.
func (sp *Scope) registerNode(node *constructorNode) (dag.NodeID, error) {
	if sp.deferAcyclicVerification {
		id, _ := sp.dag.AddUnchecked(node)
		sp.needsAcyclicCheck = true
		return id, nil
	}

	id, _, err := sp.dag.Add(node)
	if err != nil {
		if _, ok := errors.AsType[*dag.CycleError[resourceType]](err); !ok {
			return 0, err
		}
		return 0, errCycleDetected{
			Message: "cycle detected",
			Cause:   err,
		}
	}
	return id, nil
}

func (sp *Scope) checkResource(rs []resourceType) string {
	// inner
	dup := make(map[resourceType]struct{})
	for _, typ := range rs {
		if typ.groupName != "" {
			continue
		}
		if _, ok := dup[typ]; ok {
			return fmt.Sprintf("duplicate result type %s", typ)
		}
		dup[typ] = struct{}{}
	}

	// scope
	for _, typ := range rs {
		if typ.groupName != "" {
			continue
		}
		if len(sp.dag.Provide(typ)) > 0 {
			return fmt.Sprintf("result type %s has already been added", typ)
		}
	}

	return ""
}

var (
	_ func(string) Option[*resourceOptions]               = Name
	_ func(string) Option[*resourceOptions]               = Group
	_ func(uintptr) Option[*resourceOptions]              = LocationForPC
	_ func(...any) Option[*resourceOptions]               = As
	_ func(*ResourceReceiptInfo) Option[*resourceOptions] = ResourceReceipt
)

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

var (
	_ fmt.Stringer = (*ParamInfo)(nil)
	_ fmt.Stringer = (*ResultInfo)(nil)

	_ func([]*dot.Param) []*ParamInfo   = paramInfos
	_ func([]*dot.Result) []*ResultInfo = resultInfos
)

func paramInfos(dotParams []*dot.Param) []*ParamInfo {
	infos := make([]*ParamInfo, len(dotParams))
	for i, p := range dotParams {
		infos[i] = &ParamInfo{
			valueType: p.ValueType,
			optional:  p.Optional,
			valueName: p.ValueName,
			groupName: p.GroupName,
		}
	}
	return infos
}

func resultInfos(dotResults []*dot.Result) []*ResultInfo {
	infos := make([]*ResultInfo, len(dotResults))
	for i, r := range dotResults {
		infos[i] = &ResultInfo{
			valueType: r.ValueType,
			valueName: r.ValueName,
			groupName: r.GroupName,
		}
	}
	return infos
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
