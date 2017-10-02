package kubernetes

import (
	"github.com/kubermatic/kubermatic/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"
	"k8s.io/apimachinery/pkg/util/rand"
)

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubernetesFakeProvider struct {
	clusters map[string]*kubermaticv1.Cluster // by name
	cps      map[string]provider.CloudProvider
	dcs      map[string]provider.DatacenterMeta
}

// NewKubernetesFakeProvider creates a new kubernetes provider object
func NewKubernetesFakeProvider(
	dc string,
	cps map[string]provider.CloudProvider,
	dcs map[string]provider.DatacenterMeta,
) provider.ClusterProvider {
	return &kubernetesFakeProvider{
		clusters: map[string]*kubermaticv1.Cluster{
			"234jkh24234g": {
				ObjectMeta: metav1.ObjectMeta{
					Name: "234jkh24234g",
					UID:  "4711",
				},
				Spec: kubermaticv1.ClusterSpec{
					HumanReadableName: "sttts",
					MasterVersion:     "0.0.1",
					Cloud: &kubermaticv1.CloudSpec{
						DatacenterName: "fake-fra1",
						Fake: &kubermaticv1.FakeCloudSpec{
							Token: "983475982374895723958",
						},
					},
				},
				Address: &kubermaticv1.ClusterAddress{
					URL:          "http://104.155.80.128:8888",
					AdminToken:   "14c5c6cdd8bed3c849e10fc8ff1ba91571f4e06f",
					KubeletToken: "14c5c6cdd8bed3c849e10fc8ff1ba91571f4e06f",
				},
				Status: kubermaticv1.ClusterStatus{
					Phase: kubermaticv1.RunningClusterStatusPhase,
					Health: &kubermaticv1.ClusterHealth{
						LastTransitionTime: metav1.Now(),
						ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{
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

func (p *kubernetesFakeProvider) InitiateClusterUpgrade(user auth.User, name, version string) (*kubermaticv1.Cluster, error) {
	return nil, nil
}

func (p *kubernetesFakeProvider) Country() string {
	return "Germany"
}

func (p *kubernetesFakeProvider) NewClusterWithCloud(user auth.User, spec *kubermaticv1.ClusterSpec) (*kubermaticv1.Cluster, error) {
	return p.clusters["234jkh24234g"], nil
}

func (p *kubernetesFakeProvider) Cluster(user provider.User, cluster string) (*kubermaticv1.Cluster, error) {
	if _, found := p.clusters[cluster]; !found {
		return nil, errors.NewNotFound("cluster", cluster)
	}

	c := p.clusters[cluster]

	return c, nil
}

func (p *kubernetesFakeProvider) SetCloud(user auth.User, cluster string, cloud *kubermaticv1.CloudSpec) (*kubermaticv1.Cluster, error) {
	c, err := p.Cluster(user, cluster)
	if err != nil {
		return nil, err
	}
	c.Spec.Cloud = cloud
	return c, nil
}

func (p *kubernetesFakeProvider) Clusters(user auth.User) (*kubermaticv1.ClusterList, error) {
	res := &kubermaticv1.ClusterList{}
	res.Items = make([]kubermaticv1.Cluster, 0, len(p.clusters))
	for _, c := range p.clusters {
		res.Items = append(res.Items, *c)
	}

	return res, nil
}

func (p *kubernetesFakeProvider) DeleteCluster(user auth.User, cluster string) error {
	if _, found := p.clusters[cluster]; !found {
		return errors.NewNotFound("cluster", cluster)
	}

	delete(p.clusters, cluster)
	return nil
}

func (p *kubernetesFakeProvider) Nodes(user auth.User, cluster string) ([]string, error) {
	return []string{}, nil
}
