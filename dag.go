package dig

import (
	"fmt"
	"reflect"

	_ "github.com/n24-x/dag-go"
)

// DagID uniquely identifies a constructor within a DAG.
//
// It is assigned by [dag.Add] and is 1-based; 0 is reserved as the "not found"
// sentinel.
type DagID int

// resourceType describes a required or supplied resource of a node in DAG.
type resourceType struct {
	valueType reflect.Type

	// Only one of valueName or groupName will be set.
	valueName string
	groupName string
}

var _ fmt.Stringer = resourceType{}

func (k resourceType) String() string {
	if k.valueName != "" {
		return fmt.Sprintf("%v[name=%q]", k.valueType, k.valueName)
	}
	if k.groupName != "" {
		return fmt.Sprintf("%v[group=%q]", k.valueType, k.groupName)
	}
	return k.valueType.String()
}
