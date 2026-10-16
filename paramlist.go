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
	_ interface{ fmt.Stringer }             = (*paramList)(nil)
	_ interface{ Requires() []resourceKey } = (*paramList)(nil)
	_ interface{ DotParams() []*dot.Param } = (*paramList)(nil)
	_ func(reflect.Type) (paramList, error) = newParamList
)

func (pl paramList) String() string {
	params := make([]string, len(pl.Params))
	for i, p := range pl.Params {
		params[i] = p.String()
	}
	return fmt.Sprint(params)
}

func (pl paramList) Requires() []resourceKey {
	var keys []resourceKey
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

	for i := 0; i < numParameters; i++ {
		p, err := newParam(ctype.In(i))
		if err != nil {
			return pl, newErrInvalidInput(fmt.Sprintf("bad argument %d", i+1), err)
		}
		pl.Params[i] = p
	}
	return pl, nil
}
