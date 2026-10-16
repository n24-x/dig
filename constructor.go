package dig

import (
	"reflect"

	"github.com/n24-x/dig/internal/digreflect"
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
