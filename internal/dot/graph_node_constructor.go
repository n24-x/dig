package dot

// ConstructorID uniquely identifies a constructor by its function pointer.
// It is used as the key for constructor lookups in the dependency graph.
type ConstructorID uintptr

// Constructor encodes a constructor provided to the container for the DOT graph.
type Constructor struct {
	ID      ConstructorID // Unique constructor identifier.
	Name    string        // Constructor name, such as "NewHandler".
	Package string        // Package path, such as "github.com/n24-x/dig"
	File    string        // Source file path.
	Line    int           // Source line number.

	// Params is the list of regular input dependencies of this constructor.
	// Each Param represents a single dependency matched by type and optionally by name.
	Params []*Param

	// GroupParams is the list of value group input dependencies of this constructor.
	// Each Group represents a collection of values sharing the same type and group name.
	GroupParams []*Group

	// Results is the list of values produced by this constructor.
	Results []*Result

	ErrorType ErrorType // Constructor error state.
}

func (c *Constructor) removeParam(k nodeKey) {
	var pruned []*Param
	for _, p := range c.Params {
		if k != p.nodeKey() {
			pruned = append(pruned, p)
		}
	}

	c.Params = pruned
}
