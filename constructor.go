package dig

import (
	"fmt"
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"
	"github.com/n24-x/dig/internal/dot"

	dag "github.com/n24-x/dag-go"
)

type constructorNode struct {
	constructor any
	ctype       reflect.Type

	location *digreflect.Func

	constructorID dot.ConstructorID

	paramList  paramList
	resultList resultList

	// TODO
	// called bool、callback/beforeCallback
}

var (
	_ dag.Node[resourceKey]                    = (*constructorNode)(nil)
	_ fmt.Stringer                             = (*constructorNode)(nil)
	_ interface{ Location() *digreflect.Func } = (*constructorNode)(nil)
	_ interface{ ParamList() paramList }       = (*constructorNode)(nil)
	_ interface{ ResultList() resultList }     = (*constructorNode)(nil)
	_ interface{ ID() dot.ConstructorID }      = (*constructorNode)(nil)
	_ interface{ CType() reflect.Type }        = (*constructorNode)(nil)
)

var _ func(any, constructorOptions) (*constructorNode, error) = newConstructorNode

type constructorOptions struct {
	ResultName  string
	ResultGroup string
	ResultAs    []any
	Location    *digreflect.Func

	// TODO
	// Callback
	// BeforeCallback
}

// newConstructorNode parses ctor into a dag node.
//
// ctor MUST have been validated by the caller.
func newConstructorNode(ctor any, opts constructorOptions) (*constructorNode, error) {
	cval := reflect.ValueOf(ctor)
	ctype := cval.Type()
	cptr := cval.Pointer()

	params, err := newParamList(ctype)
	if err != nil {
		return nil, err
	}

	results, err := newResultList(
		ctype,
		resultOptions{
			Name:  opts.ResultName,
			Group: opts.ResultGroup,
			As:    opts.ResultAs,
		},
	)
	if err != nil {
		return nil, err
	}

	location := opts.Location
	if location == nil {
		location = digreflect.InspectFunc(ctor)
	}

	cn := &constructorNode{
		constructor:   ctor,
		ctype:         ctype,
		location:      location,
		constructorID: dot.ConstructorID(cptr),
		paramList:     params,
		resultList:    results,
	}
	return cn, nil
}

func (cn *constructorNode) Requires() []resourceKey {
	return cn.paramList.Requires()
}

func (cn *constructorNode) Provides() []resourceKey {
	return cn.resultList.Provides()
}

func (cn *constructorNode) Location() *digreflect.Func { return cn.location }
func (cn *constructorNode) ParamList() paramList       { return cn.paramList }
func (cn *constructorNode) ResultList() resultList     { return cn.resultList }
func (cn *constructorNode) ID() dot.ConstructorID      { return cn.constructorID }
func (cn *constructorNode) CType() reflect.Type        { return cn.ctype }

func (cn *constructorNode) String() string {
	return fmt.Sprintf("deps: %v, ctor: %v", cn.paramList, cn.ctype)
}
