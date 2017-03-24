package version

import (
	"fmt"

	"github.com/Masterminds/semver"
	"github.com/golang/glog"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/controller/version/dijkstra"
)

// UpdatePathSearch represents a query for a path between K8s versions
type UpdatePathSearch struct {
	updates []api.MasterUpdate
	nodes   map[string]*node
	matcher Matcher
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

// Matcher is an interface to find matches within a pattern
type Matcher interface {
	Match(pattern string, version string) bool
	Lower(a, b string) bool
}

// SemverMatcher implements Matcher for matching Semantic Versions
type SemverMatcher struct{}

// Match checks if a provided version matches the provided pattern
func (m SemverMatcher) Match(pattern string, version string) bool {
	v, err := semver.NewVersion(version)
	if err != nil {
		glog.Warningf("invalid version %q: %v", version, err)
		return false
	}

	matches, err := semver.NewConstraint(pattern)
	if err != nil {
		glog.Warningf("invalid semver pattern %q: %v", pattern, err)
		return false
	}

	return matches.Check(v)
}

// Lower determines if SemVer a is less than SemVer b
func (m SemverMatcher) Lower(a, b string) bool {
	v1, err := semver.NewVersion(a)
	if err != nil {
		glog.Warningf("invalid version %q: %v", a, err)
		return false
	}

	v2, err := semver.NewVersion(b)
	if err != nil {
		glog.Warningf("invalid version %q: %v", b, err)
		return false
	}

	return v1.Compare(v2) == -1
}

// EqualityMatcher represents a Matcher to see if strings are equal
type EqualityMatcher struct{}

// Match determines if pattern is equal to version
func (m EqualityMatcher) Match(pattern string, version string) bool { return pattern == version }

// Lower determines if a is less than b
func (m EqualityMatcher) Lower(a, b string) bool { return a < b }

// NewUpdatePathSearch finds a path between MasterVersions
func NewUpdatePathSearch(versions map[string]*api.MasterVersion, updates []api.MasterUpdate, matcher Matcher) *UpdatePathSearch {
	result := &UpdatePathSearch{
		updates: updates,
		nodes:   map[string]*node{},
		matcher: matcher,
	}

	for id, v := range versions {
		result.nodes[id] = &node{version: v}
	}

	for _, u := range updates {
		froms := []*node{}
		for id, v := range result.nodes {
			if matcher.Match(u.From, id) {
				froms = append(froms, v)
			}
		}

		tos := []*node{}
		for id, v := range result.nodes {
			if u.To == id {
				tos = append(tos, v)
			}
		}

		for _, from := range froms {
			for _, to := range tos {
				if !matcher.Lower(from.version.ID, to.version.ID) {
					continue
				}

				from.edges = append(from.edges, &edge{&u, to})
			}
		}
	}

	return result
}

// Search finds an MasterUpdate path between versions
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
