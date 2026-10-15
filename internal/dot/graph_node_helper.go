package dot

import "reflect"

// Node is a helper struct embedded into [Param] and [Result] to help them become
// nodes in a Graph.
//
// It is not itself a node in the graph.
type Node struct {
	ValueType reflect.Type

	// ValueName is not the name of the Node itself, but the attribute of the
	// object (Param or Result) that embeds this Node.
	ValueName string
	GroupName string
}

type nodeKey struct {
	valueType reflect.Type
	valueName string
	groupName string
}

func (n *Node) nodeKey() nodeKey {
	return nodeKey{valueType: n.ValueType, valueName: n.ValueName, groupName: n.GroupName}
}
