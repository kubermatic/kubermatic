package fake

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"

	"github.com/sttts/kubermatik-api/cloud"
)

type spec struct {
	nodes int
}

func NewSpec(nodes int) cloud.ClusterSpec {
	return &spec{nodes}
}

var (
	clusters map[string]cloud.Cluster = map[string]cloud.Cluster{}
	mu       sync.Mutex               // protects fields above
)

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

	nodes := make([]cloud.Node, 0, spec.nodes)
	for i := 0; i < spec.nodes; i++ {
		n := &node{
			id:       fmt.Sprintf("%s-%d", id, i),
			publicIP: "10.0.0.1",
		}

		nodes = append(nodes, n)
	}

	c := &cluster{
		id:    id,
		nodes: nodes,
	}

	clusters[id] = c
	return c, nil
}

func (p *provider) Clusters() ([]cloud.Cluster, error) {
	mu.Lock()
	defer mu.Unlock()

	cs := make([]cloud.Cluster, 0, len(clusters))
	for _, c := range clusters {
		cs = append(cs, c)
	}

	return cs, nil
}

type cluster struct {
	id    string
	nodes []cloud.Node
}

func (c *cluster) ID() string {
	return c.id
}

func (c *cluster) Nodes() []cloud.Node {
	return c.nodes
}

type node struct {
	id       string
	publicIP string
}

func (n *node) ID() string {
	return n.id
}

func (n *node) PublicIP() string {
	return n.publicIP
}

func (n *node) String() string {
	return n.id
}

func uuid() (string, error) {
	b := make([]byte, 2)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%X", b[0:2]), nil
}
