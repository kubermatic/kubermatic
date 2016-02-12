package kubernetes

import (
	"fmt"
	"sync"
	"time"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
)

var _ provider.KubernetesProvider = (*kubernetesFakeProvider)(nil)

type kubernetesFakeProvider struct {
	mu       sync.Mutex
	clusters map[string]api.Cluster // by name
	cps      map[string]provider.CloudProvider
}

// NewKubernetesFakeProvider creates a new kubernetes provider object
func NewKubernetesFakeProvider(dc string, cps map[string]provider.CloudProvider) provider.KubernetesProvider {
	return &kubernetesFakeProvider{
		clusters: map[string]api.Cluster{
			"234jkh24234g": {
				Metadata: api.Metadata{
					Name:     "234jkh24234g",
					Revision: "42",
					UID:      "4711",
					User:     "sttts",
				},
				Spec: api.ClusterSpec{
					HumanReadableName: "sttts",
					Cloud: &api.CloudSpec{
						Fake: &api.FakeCloudSpec{
							Token:  "983475982374895723958",
							Region: "fra",
							Dc:     "1",
						},
					},
				},
				Address: &api.ClusterAddress{
					URL:   "http://104.155.80.128:8888",
					Token: "14c5c6cdd8bed3c849e10fc8ff1ba91571f4e06f",
				},
				Status: api.ClusterStatus{
					Phase: api.RunningClusterStatusPhase,
					Health: &api.ClusterHealth{
						LastTransitionTime: time.Now().Add(-7 * time.Second),
						ClusterHealthStatus: api.ClusterHealthStatus{
							Apiserver:  true,
							Scheduler:  true,
							Controller: false,
							Etcd:       []bool{true},
						},
					},
				},
			},
		},
		cps: cps,
	}
}

func (p *kubernetesFakeProvider) Spec() *api.DatacenterSpec {
	return &api.DatacenterSpec{
		Description: "Fakehausen",
		Country:     "us",
		Provider:    "fake",
	}
}

func (p *kubernetesFakeProvider) Country() string {
	return "Germany"
}

func (p *kubernetesFakeProvider) NewCluster(user, cluster string, spec *api.ClusterSpec) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id, err := provider.UUID()
	if err != nil {
		return nil, err
	}

	if _, found := p.clusters[cluster]; found {
		return nil, fmt.Errorf("cluster %s already exists", cluster)
	}

	c := api.Cluster{
		Metadata: api.Metadata{
			Name:     cluster,
			Revision: "0",
			UID:      id,
		},
		Spec: *spec,
	}

	p.clusters[cluster] = c
	return &c, nil
}

func (p *kubernetesFakeProvider) Cluster(user, cluster string) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, found := p.clusters[cluster]; !found {
		return nil, kerrors.NewNotFound("cluster", cluster)
	}

	c := p.clusters[cluster]

	return &c, nil
}

func (p *kubernetesFakeProvider) Clusters(user string) ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cs := make([]*api.Cluster, 0, len(p.clusters))
	for _, c := range p.clusters {
		cs = append(cs, &c)
	}

	return cs, nil
}

func (p *kubernetesFakeProvider) DeleteCluster(user, cluster string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, found := p.clusters[cluster]; !found {
		return kerrors.NewNotFound("cluster", cluster)
	}

	delete(p.clusters, cluster)
	return nil
}

func (p *kubernetesFakeProvider) Nodes(user, cluster string) ([]string, error) {
	return []string{}, nil
}
