package dig

import (
	"fmt"
	"reflect"
	"time"

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

	observer Observer
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

var _ func(any, ConstructorOptions) (*constructorNode, error) = newConstructorNode

// newConstructorNode parses ctor into a dag node.
func newConstructorNode(ctor any, opts ConstructorOptions) (*constructorNode, error) {
	cval := reflect.ValueOf(ctor)
	ctype := cval.Type()
	cptr := cval.Pointer()

	resultList, err := newResultList(ctype, opts.result)
	if err != nil {
		return nil, err
	}

	if len(resultList.Provides()) == 0 {
		return nil, newErrInvalidInput(
			fmt.Sprintf("constructor %v must provide at least one non-error type result", ctype), nil)
	}

	paramList, err := newParamList(ctype)
	if err != nil {
		return nil, err
	}

	location := opts.location
	if location == nil {
		location = digreflect.InspectFunc(ctor)
	}

	cn := &constructorNode{
		constructor: ctor,
		ctype:       ctype,
		location:    location,
		dotID:       dot.ConstructorID(cptr),
		paramList:   paramList,
		resultList:  resultList,
	}
	return cn, nil
}

func (cn *constructorNode) String() string {
	return fmt.Sprintf("[%v]", cn.location)
}

func (cn *constructorNode) Requires() []resourceType {
	return cn.paramList.Requires().required
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
func (cn *constructorNode) Run(sp *Scope) (err error) {
	if cn.called {
		return nil
	}

	// The dependencies have already been built, so assembling the args now is
	// just a read from the scope.
	args, err := cn.paramList.Args(sp)
	if err != nil {
		return errArgumentsFailed{Func: cn.location, Reason: err}
	}

	ob := cn.observer
	obName := cn.location.Dotted()
	if ob.Announce != nil {
		ob.Announce(obName)
	}

	start := time.Now()
	if ob.Report != nil {
		defer func() {
			ob.Report(ObserveInfo{
				Name:        obName,
				Error:       err,
				ElapsedTime: time.Since(start),
			})
		}()
	}

	// Registered after Args so it only covers the call itself, and last of the
	// deferred hooks: the callback defer will go above it and rely on this one
	// running first (LIFO) to see the PanicError.
	if sp.recoverFromPanics {
		defer func() {
			if p := recover(); p != nil {
				err = PanicError{Func: cn.location, Panic: p}
			}
		}()
	}

	staging := newStagingValues()
	results := sp.fnCaller(reflect.ValueOf(cn.constructor), args)
	if err = cn.resultList.StoreResults(staging, false, results); err != nil {
		return errConstructorFailed{
			Func:   cn.location,
			Reason: err,
		}
	}

	staging.Commit(sp)
	cn.called = true
	return nil
}
