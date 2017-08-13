package kubernetes

import (
	"fmt"
	"sync"
	"time"

	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/extensions"
	"github.com/kubermatic/kubermatic/api/provider"
	"github.com/kubermatic/kubermatic/api/uuid"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/pkg/apis/rbac"
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
							Etcd:       []bool{true},
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

func (p *kubernetesFakeProvider) Country() string {
	return "Germany"
}

func (p *kubernetesFakeProvider) NewClusterWithCloud(user provider.User, spec *api.ClusterSpec, cloud *api.CloudSpec) (*api.Cluster, error) {
	spec.Cloud = cloud
	c, err := p.NewCluster(user, spec)
	if err != nil {
		return nil, err
	}

	return p.SetCloud(user, c.Metadata.Name, cloud)
}

func (p *kubernetesFakeProvider) NewCluster(user provider.User, spec *api.ClusterSpec) (*api.Cluster, error) {
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

	dc, found := p.dcs[spec.Cloud.Region]
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

func (p *kubernetesFakeProvider) Cluster(user provider.User, cluster string) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, found := p.clusters[cluster]; !found {
		return nil, errors.NewNotFound(rbac.Resource("cluster"), cluster)
	}

	c := p.clusters[cluster]

	return c, nil
}

func (p *kubernetesFakeProvider) SetCloud(user provider.User, cluster string, cloud *api.CloudSpec) (*api.Cluster, error) {
	c, err := p.Cluster(user, cluster)
	if err != nil {
		return nil, err
	}
	c.Spec.Cloud = cloud
	return c, nil
}

func (p *kubernetesFakeProvider) Clusters(user provider.User) ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	cs := make([]*api.Cluster, 0, len(p.clusters))
	for _, c := range p.clusters {
		cs = append(cs, c)
	}

	return cs, nil
}

func (p *kubernetesFakeProvider) DeleteCluster(user provider.User, cluster string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, found := p.clusters[cluster]; !found {
		return errors.NewNotFound(rbac.Resource("cluster"), cluster)
	}

	delete(p.clusters, cluster)
	return nil
}

func (p *kubernetesFakeProvider) Nodes(user provider.User, cluster string) ([]string, error) {
	return []string{}, nil
}

func (p *kubernetesFakeProvider) CreateAddon(user provider.User, cluster string, addonName string) (*extensions.ClusterAddon, error) {
	return nil, nil
}

func (p *kubernetesFakeProvider) CreateNode(user provider.User, cluster string, node *api.Node) (*extensions.ClNode, error) {
	return nil, nil
}

func (p *kubernetesFakeProvider) DeleteNode(user provider.User, cluster string, node *api.Node) error {
	return nil
}
