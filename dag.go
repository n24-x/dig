package dig

import (
	"fmt"
	"reflect"
)

// resourceType identifies one required or supplied resource of a node.
// The DAG expands each required resource into an edge per provider.
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
