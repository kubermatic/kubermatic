// Copyright (c) 2014 Flemming Andersen.
// Distributed under the MIT License (MIT)
// which can be found in the LICENSE file.

package dijkstra

import (
	"container/heap"
	"errors"
)

// Node is an interface for your own implementation of a vertex in a graph
type Node interface {
	Edges() []Edge
}

// Edge is an interface for your own implementation of an edge between two vertices in a graph
type Edge interface {
	Destination() Node
	Weight() float64
}

type NodeEdge struct {
	Node Node
	Edge Edge
}

// ShortestPath finds a shortest path between the start and end nodes.
// The two nodes' underlying values must be pointers
func ShortestPath(start, end Node) ([]NodeEdge, error) {
	visited := make(map[Node]bool)
	dists := make(map[Node]float64)
	prev := make(map[Node]*NodeEdge)

	dists[start] = 0
	queue := &queue{&queueItem{value: start, weight: 0, index: 0}}
	heap.Init(queue)

	for queue.Len() > 0 {
		// Done
		if visited[end] {
			break
		}

		item := heap.Pop(queue).(*queueItem)
		n := item.value
		for _, edge := range n.Edges() {
			dest := edge.Destination()
			dist := dists[n] + edge.Weight()
			if tentativeDist, ok := dists[dest]; !ok || dist < tentativeDist {
				dists[dest] = dist
				prev[dest] = &NodeEdge{n, edge}
				heap.Push(queue, &queueItem{value: dest, weight: dist})
			}
		}
		visited[n] = true
	}

	if !visited[end] {
		return nil, errors.New("no shortest path exists")
	}

	path := []NodeEdge{{Node: end}}
	for next := prev[end]; next != nil; next = prev[next.Node] {
		path[len(path) - 1].Edge = next.Edge
		path = append(path, *next)
	}
	// throw away first node, it has not edge
	path = path[0:len(path) - 1]

	// Reverse path
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return path, nil
}

type queueItem struct {
	value  Node
	weight float64
	index  int
}

type queue []*queueItem

func (q queue) Len() int {
	return len(q)
}

func (q queue) Less(i, j int) bool {
	return q[i].weight < q[j].weight
}

func (q queue) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
	q[i].index = i
	q[j].index = j
}

func (q *queue) Push(x interface{}) {
	n := len(*q)
	item := x.(*queueItem)
	item.index = n
	*q = append(*q, item)
}

func (q *queue) Pop() interface{} {
	old := *q
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*q = old[0 : n-1]
	return item
}

func (q *queue) update(item *queueItem, weight float64) {
	item.weight = weight
	heap.Fix(q, item.index)
}
