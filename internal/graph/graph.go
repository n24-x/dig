// Copyright (c) 2021 Uber Technologies, Inc.
// Copyright (c) 2026 k2 <skrik2@outlook.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package graph

// Graph represents a simple interface for representation
// of a directed graph.
// It is assumed that each node in the graph is uniquely
// identified with an incremental positive integer (i.e. 1, 2, 3...).
// A value of 0 for a node represents a sentinel error value.
type Graph interface {
	// Count returns the total number of nodes in the graph.
	Count() int

	// OutNeighbors returns a list of integers where each
	// represents a node that has an edge from node u.
	OutNeighbors(u int) []int
}

// Node colors used by the DFS three-color algorithm.
const (
	white = iota // Node has not been visited.
	gray         // Node is currently on the DFS recursion stack.
	black        // No cycle found from this node.
)

// IsAcyclic uses DFS with three-color marking to detect cycles.
// It returns true if the graph is acyclic.
// Otherwise, it returns false along with the first detected cycle.
func IsAcyclic(g Graph) (bool, []int) {
	state := make([]int, g.Count()+1)

	for i := 1; i <= g.Count(); i++ {
		// Start a DFS from each unvisited node to cover disconnected graphs.
		if state[i] == white {
			cycle := detectCycle(g, i, state, nil /* cycle path */)
			if len(cycle) > 0 {
				return false, cycle
			}
		}
	}

	return true, nil
}

// detectCycle performs a DFS starting from u.
// It returns the first cycle found, or nil if no cycle is reachable from u.
func detectCycle(g Graph, u int, state []int, path []int) []int {
	// Mark the current node as being explored and add it to the current path.
	state[u] = gray
	path = append(path, u)

	// Explore all outgoing neighbors.
	for _, v := range g.OutNeighbors(u) {
		switch state[v] {
		case white:
			// Continue DFS from an unvisited neighbor.
			if cycle := detectCycle(g, v, state, path); len(cycle) > 0 {
				return cycle
			}

		case gray:
			// Found a back edge, which indicates a cycle.
			for i := len(path) - 1; i >= 0; i-- {
				if path[i] == v {
					cycleLen := len(path) - i
					result := make([]int, cycleLen+1)

					// Copy the cycle into a new slice.
					copy(result, path[i:])

					// Close the cycle by appending the starting node.
					result[cycleLen] = v
					return result
				}
			}
		}
		// Black nodes have already been fully processed.
	}

	// Mark the current node as fully processed.
	state[u] = black
	return nil
}
