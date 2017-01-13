package cluster

import (
	"fmt"

	"github.com/golang/glog"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/update/dijkstra"
)

type PathSearch struct {
	updates []*api.MasterUpdate
	nodes   map[string]*node
}

type node struct {
	version *api.MasterVersion
	edges   []dijkstra.Edge
}

type edge struct {
	update *api.MasterUpdate
	dest   *node
}

func (n *node) Edges() []dijkstra.Edge {
	return n.edges
}

func (e *edge) Destination() dijkstra.Node {
	return e.dest
}

func (e *edge) Weight() float64 {
	return 1.0
}

func NewPathSearch(versions []*api.MasterVersion, updates []*api.MasterUpdate) *PathSearch {
	result := &PathSearch{
		updates: updates,
		nodes:   map[string]*node{},
	}

	for _, v := range versions {
		result.nodes[v.ID] = &node{version: v}
	}

	for _, u := range updates {
		from, found := result.nodes[u.From]
		if !found {
			glog.Warningf("Source version %q not found for update %q -> %q", u.From, u.From, u.To)
			continue
		}

		to, found := result.nodes[u.To]
		if !found {
			glog.Warningf("Destination version %q not found for update %q -> %q", u.To, u.From, u.To)
			continue
		}

		from.edges = append(from.edges, &edge{u, to})
	}

	return result
}

func (s *PathSearch) Search(from, to string) ([]*api.MasterUpdate, error) {
	fromNode, found := s.nodes[from]
	if !found {
		return nil, fmt.Errorf("source version %q not found", from)
	}

	toNode, found := s.nodes[to]
	if !found {
		return nil, fmt.Errorf("destination version %q not found", to)
	}

	p, err := dijkstra.ShortestPath(fromNode, toNode)
	if err != nil {
		return nil, err
	}

	result := make([]*api.MasterUpdate, 0, len(p))
	prev := from
	for _, ne := range p {
		v := ne.Node.(*node)
		u := ne.Edge.(*edge)
		update := *u.update
		update.From = prev
		update.To = v.version.ID
		result = append(result, &update)
		prev = v.version.ID
	}

	return result, nil
}
