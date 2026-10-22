package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/dot"
)

type paramList struct {
	ctype  reflect.Type
	Params []param
}

var (
	_ interface {
		fmt.Stringer
		Requires() []resourceType
		DotParams() []*dot.Param
	} = paramList{}

	_ func(reflect.Type) (paramList, error) = newParamList
)

func (pl paramList) String() string {
	params := make([]string, len(pl.Params))
	for i, p := range pl.Params {
		params[i] = p.String()
	}
	return fmt.Sprint(params)
}

func (pl paramList) Requires() []resourceType {
	var keys []resourceType
	for _, p := range pl.Params {
		keys = append(keys, p.Requires()...)
	}
	return keys
}

func (pl paramList) DotParams() []*dot.Param {
	var dotParams []*dot.Param
	for _, p := range pl.Params {
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
		Params: make([]param, numParameters),
	}
	for i := range numParameters {
		typ := ctype.In(i)
		p, err := newParam(typ)
		if err != nil {
			return pl, newErrInvalidInput(fmt.Sprintf("bad argument %d (%v)", i+1, typ), err)
		}
		pl.Params[i] = p
	}
	return pl, nil
}
