package dig

import (
	"fmt"
	"reflect"
)

// resourceID uniquely identifies a node in the dag.
type resourceID struct {
	valueType reflect.Type

	// Only one of valueName or groupName will be set.
	valueName string
	groupName string
}

var _ fmt.Stringer = resourceID{}

func (k resourceID) String() string {
	if k.valueName != "" {
		return fmt.Sprintf("%v[name=%q]", k.valueType, k.valueName)
	}
	if k.groupName != "" {
		return fmt.Sprintf("%v[group=%q]", k.valueType, k.groupName)
	}
	return k.valueType.String()
}
