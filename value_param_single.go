package dig

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/n24-x/dig/internal/dot"
)

type valueParam struct {
	valueType reflect.Type
	valueName string
	optional  bool
}

var (
	_ func(reflect.Type) valueParam   = newValueParam
	_ interface{ key() resourceType } = valueParam{}
)

func newValueParam(typ reflect.Type) valueParam {
	return valueParam{
		valueType: typ,
	}
}

func (pm valueParam) String() string {
	// tally.Scope[optional] means optional
	// tally.Scope[optional, name="foo"] means named optional

	var opts []string
	if pm.optional {
		opts = append(opts, "optional")
	}
	if pm.valueName != "" {
		opts = append(opts, fmt.Sprintf("name=%q", pm.valueName))
	}

	if len(opts) == 0 {
		return fmt.Sprint(pm.valueType)
	}

	return fmt.Sprintf("%v[%v]", pm.valueType, strings.Join(opts, ", "))
}

// key returns the resource this param resolves to: the value type together
// with the name it was requested under.
//
// Both Requires and Build go through key so that the resource declared in the
// graph and the resource looked up at execution time cannot drift apart.
func (pm valueParam) key() resourceType {
	return resourceType{
		valueType: pm.valueType,
		valueName: pm.valueName,
	}
}

// Requires reports the resource this param depends on.
func (pm valueParam) Requires() resourceRequirements {
	var rr resourceRequirements
	if pm.optional {
		rr.optional = []resourceType{pm.key()}
		return rr
	}
	rr.required = []resourceType{pm.key()}
	return rr
}

func (pm valueParam) LookupValue(sp *Scope) (reflect.Value, error) {
	if v, ok := sp.lookupValue(pm.key()); ok {
		return v, nil
	}

	if pm.optional && len(sp.dag.Provide(pm.key())) == 0 {
		return reflect.Zero(pm.valueType), nil
	}

	return reflect.Value{}, errValueNotFound{key: pm.key()}
}

func (pm valueParam) DotParams() []*dot.Param {
	return []*dot.Param{
		{
			Node: &dot.Node{
				ValueType: pm.valueType,
				ValueName: pm.valueName,
			},
			Optional: pm.optional,
		},
	}
}
