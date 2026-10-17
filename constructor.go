package dig

import (
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"

	dag "github.com/n24-x/dag-go"
)

type constructorNode struct {
	constructor any
	ctype       reflect.Type
	location    *digreflect.Func
	paramList   paramList
	resultList  resultList

	// TODO
	// called bool、callback/beforeCallback、id dot.ConstructorID
}

var _ dag.Node[resourceKey] = constructorNode{}

func (cn constructorNode) Requires() []resourceKey {
	return cn.paramList.Requires()
}

func (cn constructorNode) Provides() []resourceKey {
	return cn.resultList.Provides()
}
