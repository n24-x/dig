package dig

import (
	"fmt"
	"reflect"
)

// TODO
func (sp *Scope) Setup(function any, opts ...Option[*SetupOptions]) error {
	var options SetupOptions
	for _, fn := range opts {
		fn(&options)
	}

	fval := reflect.ValueOf(function)
	ftype := fval.Type()
	if ftype == nil {
		return newErrInvalidInput("cannot setup nil function", nil)
	}
	if ftype.Kind() != reflect.Func {
		return newErrInvalidInput(
			fmt.Sprintf("cannot setup non-function %v (type %v)", function, ftype), nil)
	}

	funcPL, err := newParamList(ftype)
	if err != nil {
		return err
	}

	_ = funcPL

	return nil
}

type SetupOptions struct {
	Info *SetupInfo
}

type SetupInfo struct {
	Params []*ParamInfo
}
