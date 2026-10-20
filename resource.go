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

//nolint:unused // TODO
func (sp *Scope) resource(ctor any, options resourceOptions) (err error) {
	return nil
}
