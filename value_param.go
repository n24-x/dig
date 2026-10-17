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

var _ func(reflect.Type) valueParam = newValueParam

func (ps valueParam) String() string {
	// tally.Scope[optional] means optional
	// tally.Scope[optional, name="foo"] means named optional

	var opts []string
	if ps.Optional {
		opts = append(opts, "optional")
	}
	if ps.ValueName != "" {
		opts = append(opts, fmt.Sprintf("name=%q", ps.ValueName))
	}

	if len(opts) == 0 {
		return fmt.Sprint(ps.ValueType)
	}

	return fmt.Sprintf("%v[%v]", ps.ValueType, strings.Join(opts, ", "))
}

// Requires reports the resource this param depends on.
//
// Optional params are not reported: graph edges are "must build" edges, so
// an optional dependency that nobody provides must not fail resolution.
// Build is responsible for falling back to a zero value.
func (ps valueParam) Requires() []resourceKey {
	if ps.Optional {
		return nil
	}
	return []resourceKey{
		{
			valueType: ps.ValueType,
			valueName: ps.ValueName,
		},
	}
}

func (ps valueParam) DotParams() []*dot.Param {
	return []*dot.Param{
		{
			Node: &dot.Node{
				ValueType: ps.ValueType,
				ValueName: ps.ValueName,
			},
			Optional: ps.Optional,
		},
	}
}

func newValueParam(typ reflect.Type) valueParam {
	return valueParam{
		ValueType: typ,
	}
}
