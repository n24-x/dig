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

	// dotID is the constructor's function pointer, used as its key in DOT
	// bookkeeping. It is not the DAG node number; that comes from dag.Add.
	dotID dot.ConstructorID

	paramList  paramList
	resultList resultList

	called bool

	// TODO
	// callback/beforeCallback
}

var (
	_ dag.Node[resourceType] = (*constructorNode)(nil)

	_ interface {
		Location() *digreflect.Func
		ParamList() paramList
		ResultList() resultList
		DotID() dot.ConstructorID
		CType() reflect.Type
		Run(sp *Scope) error
	} = (*constructorNode)(nil)
)

var _ func(any, constructorOptions) (*constructorNode, error) = newConstructorNode

// newConstructorNode parses ctor into a dag node.
func newConstructorNode(ctor any, opts constructorOptions) (*constructorNode, error) {
	cval := reflect.ValueOf(ctor)
	ctype := cval.Type()
	cptr := cval.Pointer()

	results, err := newResultList(ctype, opts.Result)
	if err != nil {
		return nil, err
	}

	if len(results.Provides()) == 0 {
		return nil, newErrInvalidInput(
			fmt.Sprintf("constructor %v must provide at least one non-error type result", ctype), nil)
	}

	params, err := newParamList(ctype)
	if err != nil {
		return nil, err
	}

	location := opts.Location
	if location == nil {
		location = digreflect.InspectFunc(ctor)
	}

	cn := &constructorNode{
		constructor: ctor,
		ctype:       ctype,
		location:    location,
		dotID:       dot.ConstructorID(cptr),
		paramList:   params,
		resultList:  results,
	}
	return cn, nil
}

func (cn *constructorNode) String() string {
	return fmt.Sprintf("[%v]", cn.location)
}

func (cn *constructorNode) Requires() []resourceType {
	return cn.paramList.Requires().Required
}

func (cn *constructorNode) Provides() []resourceType {
	return cn.resultList.Provides()
}

func (cn *constructorNode) Location() *digreflect.Func { return cn.location }
func (cn *constructorNode) ParamList() paramList       { return cn.paramList }
func (cn *constructorNode) ResultList() resultList     { return cn.resultList }
func (cn *constructorNode) DotID() dot.ConstructorID   { return cn.dotID }
func (cn *constructorNode) CType() reflect.Type        { return cn.ctype }

// Run builds the constructor's arguments, calls it, and commits the results to
// sp. A constructor runs at most once: later calls are no-ops.
//
// err is named because the recover and callback paths (added here, below the
// body) have to rewrite the value this call returns.
func (cn *constructorNode) Run(sp *Scope) (err error) {
	if cn.called {
		return nil
	}

	args, err := cn.paramList.Args(sp)
	if err != nil {
		return errArgumentsFailed{Func: cn.location, Reason: err}
	}

	staging := newStagingValues()
	results := sp.fnCaller(reflect.ValueOf(cn.constructor), args)
	if err = cn.resultList.ExtractList(staging, results); err != nil {
		return errConstructorFailed{
			Func:   cn.location,
			Reason: err,
		}
	}

	staging.Commit(sp)
	cn.called = true
	return nil
}
