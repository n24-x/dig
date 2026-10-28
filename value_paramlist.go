package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

type paramList struct {
	ctype  reflect.Type
	params []param
}

var (
	_ interface {
		fmt.Stringer
		Requires() resourceRequirements
		DotParams() []*dot.Param
	} = paramList{}

	_ func(reflect.Type) (paramList, error) = newParamList
)

func (pl paramList) String() string {
	params := make([]string, len(pl.params))
	for i, p := range pl.params {
		params[i] = p.String()
	}
	return fmt.Sprint(params)
}

func (pl paramList) Requires() resourceRequirements {
	var rr resourceRequirements
	for _, p := range pl.params {
		requiredKeys := p.Requires().required
		optionalKeys := p.Requires().optional

		rr.required = append(rr.required, requiredKeys...)
		rr.optional = append(rr.optional, optionalKeys...)
	}
	return rr
}

func (pl paramList) DotParams() []*dot.Param {
	var dotParams []*dot.Param
	for _, p := range pl.params {
		dotParams = append(dotParams, p.DotParams()...)
	}

	return dotParams
}

func newParamList(ctype reflect.Type) (paramList, error) {
	numParameters := ctype.NumIn()
	if ctype.IsVariadic() {
		// NOTE: If the function is variadic, we skip the last argument
		// because we're not filling variadic arguments yet.
		numParameters--
	}

	pl := paramList{
		ctype:  ctype,
		params: make([]param, numParameters),
	}
	for i := range numParameters {
		typ := ctype.In(i)
		p, err := newParam(typ)
		if err != nil {
			return pl, newErrInvalidInput(fmt.Sprintf("bad argument %d (%v)", i+1, typ), err)
		}
		pl.params[i] = p
	}
	return pl, nil
}

func (pl paramList) Args(sp *Scope) ([]reflect.Value, error) {
	args := make([]reflect.Value, len(pl.params))
	for i, p := range pl.params {
		var err error
		args[i], err = p.LookupValue(sp)
		if err != nil {
			return nil, err
		}
	}
	return args, nil
}
