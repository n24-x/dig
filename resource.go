package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"
)

func (ct *Container) Resource(ctor any, opts ...Option[*resourceOptions]) error {
	if ct.rootScope == nil {
		return fmt.Errorf("dig: container has a nil root scope")
	}
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

		return errConstructor{
			Func:   badFunc,
			Reason: err,
		}
	}
	return nil
}

func (sp *Scope) resource(ctor any, options resourceOptions) (err error) {
	return nil
}

// Name sets the name of the provided value. Validation is performed once by
// [Validate] rather than here.
func Name(name string) Option[*resourceOptions] {
	return func(ro *resourceOptions) {
		if ro == nil {
			return
		}
		ro.ValueName = name
	}
}

// Group places the provided value in the given group. Validation is performed
// once by [Validate] rather than here.
func Group(group string) Option[*resourceOptions] {
	return func(ro *resourceOptions) {
		if ro == nil {
			return
		}
		ro.GroupName = group
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
