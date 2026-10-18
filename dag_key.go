package dig

import (
	"fmt"
	"reflect"
)

// resourceKey uniquely identifies a node in the dag.
type resourceKey struct {
	valueType reflect.Type

	// Only one of valueName or groupName will be set.
	valueName string
	groupName string
}

var _ fmt.Stringer = resourceKey{}

func (k resourceKey) String() string {
	if k.valueName != "" {
		return fmt.Sprintf("%v[name=%q]", k.valueType, k.valueName)
	}
	if k.groupName != "" {
		return fmt.Sprintf("%v[group=%q]", k.valueType, k.groupName)
	}
	return k.valueType.String()
}
