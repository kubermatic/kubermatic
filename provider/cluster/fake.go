package cluster

import (
	"fmt"
	"sync"

	"github.com/kubermatic/api/provider"
)

var _ provider.ClusterProvider = (*clusterProvider)(nil)

type clusterProvider struct {
	mu       sync.Mutex
	clusters map[string]map[string]provider.Cluster // by dc and name
}

func NewClusterProvider() provider.ClusterProvider {
	return &clusterProvider{
		clusters: map[string]map[string]provider.Cluster{},
	}
}

func (p *clusterProvider) NewCluster(name string, spec provider.ClusterSpec) (*provider.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id, err := provider.Uuid()
	if err != nil {
		return nil, err
	}

	if _, found := p.clusters[spec.Dc]; !found {
		p.clusters[spec.Dc] = map[string]provider.Cluster{}
	}
	if _, found := p.clusters[spec.Dc][name]; found {
		return nil, fmt.Errorf("cluster %s already exists in dc %s", name, spec.Dc)
	}

	c := provider.Cluster{
		Metadata: provider.Metadata{
			Name: name,
			Uid:  "fake-" + id,
		},
		Spec: spec,
	}
	p.clusters[spec.Dc][name] = c
	return &c, nil
}

func (p *clusterProvider) Cluster(dc string, name string) (*provider.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, found := p.clusters[dc]; !found {
		return nil, fmt.Errorf("dc %s not found", dc)
	}
	if _, found := p.clusters[dc][name]; !found {
		return nil, fmt.Errorf("cluster %s not found in dc %s", name, dc)
	}

	c := p.clusters[dc][name]
	return &c, nil
}

func (p *clusterProvider) Clusters(dc string) ([]*provider.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, found := p.clusters[dc]; !found {
		return nil, fmt.Errorf("dc %s not found", dc)
	}

	cs := make([]*provider.Cluster, len(p.clusters[dc]))
	for _, c := range p.clusters[dc] {
		cs = append(cs, &c)
	}

	return cs, nil
}
