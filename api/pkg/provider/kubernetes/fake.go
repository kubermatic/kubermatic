package kubernetes

import (
	"fmt"
	"sync"
	"time"

	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"
	"k8s.io/apimachinery/pkg/util/rand"
)

var _ provider.KubernetesProvider = (*kubernetesFakeProvider)(nil)

type kubernetesFakeProvider struct {
	mu       sync.Mutex
	clusters map[string]*api.Cluster // by name
	cps      map[string]provider.CloudProvider
	dcs      map[string]provider.DatacenterMeta
}

// NewKubernetesFakeProvider creates a new kubernetes provider object
func NewKubernetesFakeProvider(
	dc string,
	cps map[string]provider.CloudProvider,
	dcs map[string]provider.DatacenterMeta,
) provider.KubernetesProvider {
	return &kubernetesFakeProvider{
		clusters: map[string]*api.Cluster{
			"234jkh24234g": {
				Metadata: api.Metadata{
					Name:     "234jkh24234g",
					Revision: "42",
					UID:      "4711",
					User:     "sttts",
				},
				Spec: api.ClusterSpec{
					HumanReadableName: "sttts",
					MasterVersion:     "0.0.1",
					Cloud: &api.CloudSpec{
						DatacenterName: "fake-fra1",
						Fake: &api.FakeCloudSpec{
							Token: "983475982374895723958",
						},
					},
				},
				Address: &api.ClusterAddress{
					URL:          "http://104.155.80.128:8888",
					AdminToken:   "14c5c6cdd8bed3c849e10fc8ff1ba91571f4e06f",
					KubeletToken: "14c5c6cdd8bed3c849e10fc8ff1ba91571f4e06f",
				},
				Status: api.ClusterStatus{
					Phase: api.RunningClusterStatusPhase,
					Health: &api.ClusterHealth{
						LastTransitionTime: time.Now().Add(-7 * time.Second),
						ClusterHealthStatus: api.ClusterHealthStatus{
							Apiserver:  true,
							Scheduler:  true,
							Controller: false,
							Etcd:       true,
						},
					},
				},
			},
		},
		cps: cps,
		dcs: dcs,
	}
}

func (p *kubernetesFakeProvider) Spec() *api.DatacenterSpec {
	return &api.DatacenterSpec{
		Location: "Fakehausen",
		Country:  "US",
		Provider: "fake",
	}
}

func (p *kubernetesFakeProvider) UpgradeCluster(user auth.User, cluster, version string) error {
	return nil
}

func (p *kubernetesFakeProvider) Country() string {
	return "Germany"
}

func (p *kubernetesFakeProvider) NewClusterWithCloud(user auth.User, spec *api.ClusterSpec) (*api.Cluster, error) {
	c, err := p.NewCluster(user, spec)
	if err != nil {
		return nil, err
	}

	return p.SetCloud(user, c.Metadata.Name, c.Spec.Cloud)
}

func (p *kubernetesFakeProvider) NewCluster(user auth.User, spec *api.ClusterSpec) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	id, err := uuid.UUID()
	if err != nil {
		return nil, err
	}

	cluster := rand.String(9)
	if _, found := p.clusters[cluster]; found {
		return nil, fmt.Errorf("cluster %s already exists", cluster)
	}

	dc, found := p.dcs[spec.Cloud.DatacenterName]
	if !found {
		return nil, errors.NewBadRequest("Unregistered datacenter")
	}

	c := &api.Cluster{
		Metadata: api.Metadata{
			Name:     cluster,
			Revision: "0",
			UID:      id,
		},
		Spec: *spec,
		Seed: dc.Seed,
	}

	p.clusters[cluster] = c
	return c, nil
}

func (p *kubernetesFakeProvider) Cluster(user auth.User, cluster string) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, found := p.clusters[cluster]; !found {
		return nil, errors.NewNotFound("cluster", cluster)
	}

	c := p.clusters[cluster]

	return c, nil
}

func (p *kubernetesFakeProvider) SetCloud(user auth.User, cluster string, cloud *api.CloudSpec) (*api.Cluster, error) {
	c, err := p.Cluster(user, cluster)
	if err != nil {
		return nil, err
	}
	c.Spec.Cloud = cloud
	return c, nil
}

func (p *kubernetesFakeProvider) Clusters(user auth.User) ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cs := make([]*api.Cluster, 0, len(p.clusters))
	for _, c := range p.clusters {
		cs = append(cs, c)
	}

	return cs, nil
}

func (p *kubernetesFakeProvider) DeleteCluster(user auth.User, cluster string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, found := p.clusters[cluster]; !found {
		return errors.NewNotFound("cluster", cluster)
	}

	delete(p.clusters, cluster)
	return nil
}

func (p *kubernetesFakeProvider) Nodes(user auth.User, cluster string) ([]string, error) {
	return []string{}, nil
}
