package dig

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/n24-x/dig/internal/dot"
)

type valueParam struct {
	ValueType reflect.Type
	ValueName string
	Optional  bool
}

var (
	_ func(reflect.Type) valueParam   = newValueParam
	_ interface{ key() resourceType } = valueParam{}
)

func newValueParam(typ reflect.Type) valueParam {
	return valueParam{
		ValueType: typ,
	}
}

func (pm valueParam) String() string {
	// tally.Scope[optional] means optional
	// tally.Scope[optional, name="foo"] means named optional

	var opts []string
	if pm.Optional {
		opts = append(opts, "optional")
	}
	if pm.ValueName != "" {
		opts = append(opts, fmt.Sprintf("name=%q", pm.ValueName))
	}

	if len(opts) == 0 {
		return fmt.Sprint(pm.ValueType)
	}

	return fmt.Sprintf("%v[%v]", pm.ValueType, strings.Join(opts, ", "))
}

// key returns the resource this param resolves to: the value type together
// with the name it was requested under.
//
// Both Requires and Build go through key so that the resource declared in the
// graph and the resource looked up at execution time cannot drift apart.
func (pm valueParam) key() resourceType {
	return resourceType{
		valueType: pm.ValueType,
		valueName: pm.ValueName,
	}
}

// Requires reports the resource this param depends on.
func (pm valueParam) Requires() resourceRequirements {
	var rr resourceRequirements
	if pm.Optional {
		rr.Optional = []resourceType{pm.key()}
		return rr
	}
	rr.Required = []resourceType{pm.key()}
	return rr
}

// TODO
func (pm valueParam) Value(sp *Scope) (reflect.Value, error) {
	// TODO decorated

	if v, ok := sp.lookupValue(pm.key()); ok {
		return v, nil
	}

	if pm.Optional && len(sp.dag.Provide(pm.key())) == 0 {
		return reflect.Zero(pm.ValueType), nil
	}

	return reflect.Value{}, errValueNotFound{key: pm.key()}
}

func (pm valueParam) DotParams() []*dot.Param {
	return []*dot.Param{
		{
			Node: &dot.Node{
				ValueType: pm.ValueType,
				ValueName: pm.ValueName,
			},
			Optional: pm.Optional,
		},
	}
}
