package dot

import "fmt"

// Param is a parameter node in the graph. Parameters are the input to constructors.
type Param struct {
	*Node

	Optional bool
}

// DotString returns the identifier used for this node in DOT output.
func (p *Param) DotString() string {
	if p.ValueName != "" {
		return fmt.Sprintf("%v[name=%v]", p.ValueType.String(), p.ValueName)
	}
	return p.ValueType.String()
}
