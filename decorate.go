package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"
)

var _ interface {
	Decorate(dtor any, opts ...Option[*DecorateOptions]) error
} = (*Container)(nil)

var _ interface {
	Decorate(dtor any, opts ...Option[*DecorateOptions]) error
	decorate(dtor any, opts DecorateOptions) error
	registerDecorator(node *decoratorNode)
	checkDecorate(rs []resourceType) string
} = (*Scope)(nil)

var _ func(any) error = decoratorValidate

func (ct *Container) Decorate(dtor any, opts ...Option[*DecorateOptions]) error {
	return ct.rootScope.Decorate(dtor, opts...)
}

func (sp *Scope) Decorate(dtor any, opts ...Option[*DecorateOptions]) error {
	err := decoratorValidate(dtor)
	if err != nil {
		return err
	}

	var options DecorateOptions
	for _, fn := range opts {
		fn(&options)
	}

	if err := sp.decorate(dtor, options); err != nil {
		return errDecorateFailed{
			Func:   digreflect.InspectFunc(dtor),
			Reason: err,
		}
	}

	return nil
}

func (sp *Scope) decorate(dtor any, opts DecorateOptions) error {
	node, err := newDecoratorNode(dtor, sp, opts)
	if err != nil {
		return err
	}

	node.observer = opts.observer

	if reason := sp.checkDecorate(node.resultList.Provides()); reason != "" {
		return newErrInvalidInput(reason, nil)
	}

	sp.registerDecorator(node)

	if info := opts.info; info != nil {
		info.Params = paramInfos(node.paramList.DotParams())
		info.Results = resultInfos(node.resultList.DotResults())
	}
	return nil
}

func (sp *Scope) registerDecorator(node *decoratorNode) {
	types := node.resultList.Provides()
	for _, typ := range types {
		sp.decorators[typ] = node
	}
}

// decoratorValidate reports whether dtor is a usable decorator.
//
// Parsing the params and results is most of it: they reject a return struct
// where a parameter struct belongs, an error result, an unexported field and
// the tag combinations that make no sense. The one rule that only matters for
// a decorator comes after.
//
// Parameters need no rule of their own. A group parameter must already be a
// slice, and a flattened one is rejected outright, so anything that parses as
// a group parameter already names the group as a whole.
//
// A group result does need a rule: a provider may add one member to a group,
// but a decorator replaces the group, and replacing it with a single member
// leaves nothing to spread back into a slice. flatten needs no separate
// message: it keys the group by the element type, which lands on the same
// check.
func decoratorValidate(dtor any) error {
	dtype := reflect.TypeOf(dtor)
	if dtype == nil {
		return newErrInvalidInput("cannot decorate nil", nil)
	}

	if dtype.Kind() != reflect.Func {
		return newErrInvalidInput(
			fmt.Sprintf("cannot decorate non-function %v (type %v)", dtor, dtype), nil)
	}

	if _, err := newParamList(dtype); err != nil {
		return err
	}

	resultList, err := newResultList(dtype, resultOptions{})
	if err != nil {
		return err
	}

	for _, result := range resultList.results {
		for _, key := range result.Provides() {
			if key.groupName != "" && key.valueType.Kind() != reflect.Slice {
				return newErrInvalidInput(fmt.Sprintf(
					"cannot decorate group %q: a decorator replaces the group as a "+
						"whole, so its result must hold the new members", key.groupName), nil)
			}
		}
	}

	return nil
}

func (sp *Scope) checkDecorate(rs []resourceType) string {
	// self
	dup := make(map[resourceType]struct{})
	for _, typ := range rs {
		if _, ok := dup[typ]; ok {
			return fmt.Sprintf("duplicate decorate %s", typ)
		}
		dup[typ] = struct{}{}
	}

	// scope
	for _, key := range rs {
		if _, ok := sp.decorators[key]; ok {
			return fmt.Sprintf("type %s already has decorator", key)
		}
	}

	return ""
}
