package fake

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"

	"github.com/kubermatic/api/provider"
)

var (
	clusters map[string]provider.Cluster = map[string]provider.Cluster{}
	mu       sync.Mutex                  // protects fields above
)

type spec struct {
	nodes int
}

func NewSpec(nodes int) provider.ClusterSpec {
	return &spec{nodes}
}

var _ provider.ClusterProvider = (*clusterProvider)(nil)

type clusterProvider struct{}

func NewProvider() provider.ClusterProvider {
	return &clusterProvider{}
}

func (p *clusterProvider) NewCluster(s provider.ClusterSpec) (provider.Cluster, error) {
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

	nodes := make([]provider.Node, spec.nodes)
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

func (p *clusterProvider) Clusters() ([]provider.Cluster, error) {
	mu.Lock()
	defer mu.Unlock()

	cs := make([]provider.Cluster, len(clusters))
	var i int
	for _, c := range clusters {
		cs[i] = c
		i++
	}

	return cs, nil
}

var _ provider.Cluster = (*cluster)(nil)

type cluster struct {
	FakeID    string          `json:"id"`
	FakeNodes []provider.Node `json:"nodes"`
}

func (c *cluster) ID() string {
	return c.FakeID
}

func (c *cluster) Nodes() []provider.Node {
	return c.FakeNodes
}

var _ provider.Node = (*node)(nil)

type node struct {
	FakeID       string `json:"id"`
	FakePublicIP string `json:"public"`
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
