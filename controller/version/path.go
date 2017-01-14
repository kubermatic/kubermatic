package version

import (
	"fmt"

	"github.com/golang/glog"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/version/dijkstra"
)

type UpdatePathSearch struct {
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

func NewUpdatePathSearch(versions map[string]*api.MasterVersion, updates []*api.MasterUpdate) *UpdatePathSearch {
	result := &UpdatePathSearch{
		updates: updates,
		nodes:   map[string]*node{},
	}

	for id, v := range versions {
		result.nodes[id] = &node{version: v}
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

func (s *UpdatePathSearch) Search(from, to string) ([]*api.MasterUpdate, error) {
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
