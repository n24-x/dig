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
	_ func(*Container, any, ...Option[*ResourceOptions]) error = (*Container).Resource
	_ func(*Scope, any, ...Option[*ResourceOptions]) error     = (*Scope).Resource
	_ func(*Scope, any, ResourceOptions) error                 = (*Scope).resource
	_ func(*Scope, []resourceType) string                      = (*Scope).checkResource
)

func (ct *Container) Resource(ctor any, opts ...Option[*ResourceOptions]) error {
	return ct.rootScope.Resource(ctor, opts...)
}

func (sp *Scope) Resource(ctor any, opts ...Option[*ResourceOptions]) error {
	ctype := reflect.TypeOf(ctor)
	if ctype == nil {
		return newErrInvalidInput("dig: nil cannot provide resource", nil)
	}

	if ctype.Kind() != reflect.Func {
		return newErrInvalidInput(
			fmt.Sprintf("must be function, got %v (type %v)",
				ctor, ctype), nil)
	}

	var options ResourceOptions
	for _, fn := range opts {
		fn(&options)
	}
	if err := options.Validate(); err != nil {
		return err
	}

	if err := sp.resource(ctor, options); err != nil {
		var badFunc *digreflect.Func
		if options.location == nil {
			badFunc = digreflect.InspectFunc(ctor)
		} else {
			badFunc = options.location
		}

		return errResourceAddFailed{
			Func:   badFunc,
			Reason: err,
		}
	}
	return nil
}

func (sp *Scope) resource(ctor any, options ResourceOptions) error {
	node, err := newConstructorNode(ctor, ConstructorOptions{
		result: resultOptions{
			nameTag:  options.nameTag,
			groupTag: options.groupTag,
			asTypes:  options.asTypes,
		},
		location: options.location,
	})
	if err != nil {
		return err
	}

	node.observer = options.observer

	if reason := sp.checkResource(node.Provides()); reason != "" {
		return newErrInvalidInput(reason, nil)
	}

	id, err := sp.registerNode(node)
	if err != nil {
		return err
	}

	if info := options.info; info != nil {
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
		if sp.hasOwnConstructor(typ) {
			return fmt.Sprintf("result type %s has already been added", typ)
		}
	}

	return ""
}

var (
	_ func(string) Option[*ResourceOptions]               = Name
	_ func(string) Option[*ResourceOptions]               = Group
	_ func(uintptr) Option[*ResourceOptions]              = LocationForPC
	_ func(...any) Option[*ResourceOptions]               = As
	_ func(*ResourceReceiptInfo) Option[*ResourceOptions] = ResourceReceipt
	_ func(Observer) Option[*ResourceOptions]             = ResourceObserver
)

// ResourceObserver makes the container report what happens to the constructor
// passed to [Scope.Resource].
//
// Both halves are registered at once because they describe the same run.
// Passing a zero Observer is legal and observes nothing.
func ResourceObserver(ob Observer) Option[*ResourceOptions] {
	return func(ro *ResourceOptions) {
		if ro == nil {
			return
		}
		ro.observer = ob
	}
}

// Name sets the name of the provided value.
//
// TODO cleanup
// Validation is performed once by [Validate] rather than here.
func Name(nameTag string) Option[*ResourceOptions] {
	return func(ro *ResourceOptions) {
		if ro == nil {
			return
		}
		ro.nameTag = nameTag
	}
}

// Group places the provided value in the given group.
//
// TODO cleanup
// Validation is performed once by [Validate] rather than here.
func Group(groupTag string) Option[*ResourceOptions] {
	return func(ro *ResourceOptions) {
		if ro == nil {
			return
		}
		ro.groupTag = groupTag
	}
}

// LocationForPC records the call site that constructed this option. It is used
// to attribute errors to user code instead of to this package.
func LocationForPC(pc uintptr) Option[*ResourceOptions] {
	return func(ro *ResourceOptions) {
		if ro == nil {
			return
		}
		ro.location = digreflect.InspectFuncPC(pc)
	}
}

// As makes the produced value available as the given interfaces instead of its
// own type. Each argument must be a pointer to an interface that the value
// implements; because the argument is untyped, this cannot be checked by the
// compiler and is validated by [Validate] and at parse time instead.
//
// It cannot be combined with a flattened group: flatten keys the group by the
// element type of the value, while As would key it by one of the interfaces.
func As(types ...any) Option[*ResourceOptions] {
	return func(ro *ResourceOptions) {
		if ro == nil {
			return
		}
		ro.asTypes = append(ro.asTypes, types...)
	}
}

// ResourceReceipt writes the receipt of this resource registration into info.
//
// The receipt is filled only after the resource has been registered
// successfully. Passing a nil info is a no-op.
func ResourceReceipt(info *ResourceReceiptInfo) Option[*ResourceOptions] {
	return func(ro *ResourceOptions) {
		if ro == nil {
			return
		}
		ro.info = info
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
