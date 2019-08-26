package denypackages

import "golang.org/x/tools/go/callgraph"

type callback func(edge *callgraph.Edge, previous []*callgraph.Edge) (follow bool)

func GraphVisitEdges(root *callgraph.Node, callback callback) {
	stack := make([]*callgraph.Edge, 0, 32)
	visited := make(map[*callgraph.Node]bool)

	visit(root, callback, visited, stack)
}

func visit(n *callgraph.Node, callback callback, visited map[*callgraph.Node]bool, stack []*callgraph.Edge) {
	if visited[n] {
		return
	}
	visited[n] = true

	for _, edge := range n.Out {
		follow := callback(edge, stack)
		if !follow {
			continue
		}

		visit(edge.Callee, callback, visited, append(stack, edge))
	}
}
