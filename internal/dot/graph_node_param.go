package dot

import "fmt"

// Param is a parameter node in the graph. Parameters are the input to constructors.
type Param struct {
	*Node

	Optional bool
}

// String implements fmt.Stringer for Param.
func (p *Param) String() string {
	if p.ValueName != "" {
		return fmt.Sprintf("%v[name=%v]", p.ValueType.String(), p.ValueName)
	}
	return p.ValueType.String()
}
