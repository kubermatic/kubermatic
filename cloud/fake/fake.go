package fake

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"

	"github.com/sttts/kubermatic-api/cloud"
)

var (
	clusters map[string]cloud.Cluster = map[string]cloud.Cluster{}
	mu       sync.Mutex               // protects fields above
)

type spec struct {
	nodes int
}

func NewSpec(nodes int) cloud.ClusterSpec {
	return &spec{nodes}
}

type provider struct{}

func NewProvider() cloud.Provider {
	return &provider{}
}

func (p *provider) NewCluster(s cloud.ClusterSpec) (cloud.Cluster, error) {
	mu.Lock()
	defer mu.Unlock()

	spec := s.(*spec)
	if spec.nodes <= 0 {
		return nil, errors.New("illegal node count")
	}

	id, err := uuid()
	if err != nil {
		return nil, err
	}
	id = "fake-" + id

	nodes := make([]cloud.Node, spec.nodes)
	for i := 0; i < spec.nodes; i++ {
		n := &node{
			FakeID:       fmt.Sprintf("%s-%d", id, i),
			FakePublicIP: "10.0.0.1",
		}

		nodes[i] = n
	}

	c := &cluster{
		FakeID:    id,
		FakeNodes: nodes,
	}

	clusters[id] = c
	return c, nil
}

func (p *provider) Clusters() ([]cloud.Cluster, error) {
	mu.Lock()
	defer mu.Unlock()

	cs := make([]cloud.Cluster, len(clusters))
	var i int
	for _, c := range clusters {
		cs[i] = c
		i++
	}

	return cs, nil
}

type cluster struct {
	FakeID    string       `json: id`
	FakeNodes []cloud.Node `json: nodes`
}

func (c *cluster) ID() string {
	return c.FakeID
}

func (c *cluster) Nodes() []cloud.Node {
	return c.FakeNodes
}

type node struct {
	FakeID       string `json: id`
	FakePublicIP string `json: public`
}

func (n *node) ID() string {
	return n.FakeID
}

func (n *node) PublicIP() string {
	return n.FakePublicIP
}

func uuid() (string, error) {
	b := make([]byte, 2)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", b[0:2]), nil
}
