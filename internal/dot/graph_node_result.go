package dot

import "fmt"

// Result is a result node in the graph. It represents a value produced by a
// constructor but does not store the runtime value itself. Grouped results are
// additionally referenced by their corresponding [Group].
type Result struct {
	*Node

	// IndexInGroup is added to differentiate grouped values from one another.
	// Since grouped values have the same type and group, their Node / string
	// representations are the same, so we need indices to uniquely identify
	// the values.
	IndexInGroup int
}

// String implements fmt.Stringer for Result.
func (r *Result) String() string {
	switch {
	case r.ValueName != "":
		return fmt.Sprintf("%v[name=%v]", r.ValueType.String(), r.ValueName)
	case r.GroupName != "":
		return fmt.Sprintf("%v[group=%v]%v", r.ValueType.String(), r.GroupName, r.IndexInGroup)
	default:
		return r.ValueType.String()
	}
}

// Attributes composes and returns a string of the Result node's attributes.
func (r *Result) Attributes() string {
	switch {
	case r.ValueName != "":
		return fmt.Sprintf(`label=<%v<BR /><FONT POINT-SIZE="10">Name: %v</FONT>>`,
			r.ValueType,
			r.ValueName,
		)
	case r.GroupName != "":
		return fmt.Sprintf(`label=<%v<BR /><FONT POINT-SIZE="10">Group: %v</FONT>>`, r.ValueType, r.GroupName)
	default:
		return fmt.Sprintf(`label=<%v>`, r.ValueType)
	}
}
