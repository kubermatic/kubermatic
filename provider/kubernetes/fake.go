package kubernetes

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
)

var _ provider.KubernetesProvider = (*kubernetesProvider)(nil)

type kubernetesProvider struct {
	mu       sync.Mutex
	clusters map[string]map[string]api.Cluster // by dc and name
}

// NewKubernetesProvider creates a new kubernetes provider object
func NewKubernetesProvider() provider.KubernetesProvider {
	url, _ := url.Parse("http://104.155.80.128:8888")
	return &kubernetesProvider{
		clusters: map[string]map[string]api.Cluster{
			"fra-1": {
				"sttts": api.Cluster{
					Metadata: api.Metadata{
						Name:     "sttts",
						Revision: 42,
						UID:      "4711",
						Annotations: map[string]string{
							"user":                "sttts",
							"digitalocean-token":  "983475982374895723958",
							"digitalocean-region": "fra",
							"digitalocean-dc":     "1",
						},
					},
					Spec: api.ClusterSpec{
						Dc: "fra-1",
					},
					Address: &api.ClusterAddress{
						URL: *url,
					},
					Status: &api.ClusterStatus{
						Health: api.ClusterHealth{
							Timestamp:  time.Now().Add(-7 * time.Second),
							Apiserver:  true,
							Scheduler:  true,
							Controller: false,
							Etcd:       true,
						},
					},
				},
			},
		},
	}
}

func (p *kubernetesProvider) NewCluster(name string, spec api.ClusterSpec) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id, err := provider.UUID()
	if err != nil {
		return nil, err
	}

	if _, found := p.clusters[spec.Dc]; !found {
		p.clusters[spec.Dc] = map[string]api.Cluster{}
	}
	if _, found := p.clusters[spec.Dc][name]; found {
		return nil, fmt.Errorf("cluster %s already exists in dc %s", name, spec.Dc)
	}

	c := api.Cluster{
		Metadata: api.Metadata{
			Name:     name,
			Revision: 0,
			UID:      id,
		},
		Spec: spec,
	}
	p.clusters[spec.Dc][name] = c
	return &c, nil
}

func (p *kubernetesProvider) Cluster(dc string, name string) (*api.Cluster, error) {
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

func (p *kubernetesProvider) Clusters(dc string) ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, found := p.clusters[dc]; !found {
		return nil, fmt.Errorf("dc %s not found", dc)
	}

	cs := make([]*api.Cluster, len(p.clusters[dc]))
	for _, c := range p.clusters[dc] {
		cs = append(cs, &c)
	}

	return cs, nil
}

func (p *kubernetesProvider) Nodes(dc string, cluster string) ([]string, error) {
	return []string{}, nil
}
