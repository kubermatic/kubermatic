package kubernetes

import (
	"fmt"
	"sync"
	"time"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
)

var _ provider.KubernetesProvider = (*kubernetesProvider)(nil)

type kubernetesProvider struct {
	mu       sync.Mutex
	clusters map[string]api.Cluster // by name
	cps      map[string]provider.CloudProvider
}

// NewKubernetesProvider creates a new kubernetes provider object
func NewKubernetesFakeProvider(dc string, cps map[string]provider.CloudProvider) provider.KubernetesProvider {
	return &kubernetesProvider{
		clusters: map[string]api.Cluster{
			"sttts": {
				Metadata: api.Metadata{
					Name:     "sttts",
					Revision: 42,
					UID:      "4711",
					Annotations: map[string]string{
						"user":              "sttts",
						"cloud-provider":    provider.FakeCloudProvider,
						"cloud-fake-token":  "983475982374895723958",
						"cloud-fake-region": "fra",
						"cloud-fake-dc":     "1",
					},
				},
				Spec: api.ClusterSpec{},
				Address: &api.ClusterAddress{
					URL:   "http://104.155.80.128:8888",
					Token: "14c5c6cdd8bed3c849e10fc8ff1ba91571f4e06f",
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
		cps: cps,
	}
}

func (p *kubernetesProvider) NewCluster(name string, spec api.ClusterSpec) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id, err := provider.UUID()
	if err != nil {
		return nil, err
	}

	if _, found := p.clusters[name]; found {
		return nil, fmt.Errorf("cluster %s already exists", name)
	}

	c := api.Cluster{
		Metadata: api.Metadata{
			Name:     name,
			Revision: 0,
			UID:      id,
		},
		Spec: spec,
	}

	err = provider.MarshalClusterCloud(p.cps, &c)
	if err != nil {
		return nil, err
	}

	p.clusters[name] = c
	return &c, nil
}

func (p *kubernetesProvider) Cluster(name string) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, found := p.clusters[name]; !found {
		return nil, fmt.Errorf("cluster %q not found", name)
	}

	c := p.clusters[name]

	err := provider.UnmarshalClusterCloud(p.cps, &c)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (p *kubernetesProvider) Clusters() ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cs := make([]*api.Cluster, 0, len(p.clusters))
	for _, c := range p.clusters {
		err := provider.UnmarshalClusterCloud(p.cps, &c)
		if err != nil {
			return nil, err
		}

		cs = append(cs, &c)
	}

	return cs, nil
}

func (p *kubernetesProvider) Nodes(cluster string) ([]string, error) {
	return []string{}, nil
}
