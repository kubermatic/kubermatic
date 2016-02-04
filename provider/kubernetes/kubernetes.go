package kubernetes

import (
	"errors"
	"log"
	"sync"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"time"
)

var _ provider.KubernetesProvider = (*kubernetesProvider)(nil)

type kubernetesProvider struct {
	mu     sync.Mutex
	cps    map[string]provider.CloudProvider
	client *unversioned.Client

	description, country, provider string
}

// NewKubernetesProvider creates a new kubernetes provider object
func NewKubernetesProvider(
	clientConfig *unversioned.Config,
	cps map[string]provider.CloudProvider,
	desc, country, provider string,
) provider.KubernetesProvider {
	client, err := unversioned.New(clientConfig)
	if err != nil {
		log.Fatal(err)
	}

	return &kubernetesProvider{
		cps:         cps,
		client:      client,
		description: desc,
		country:     country,
		provider:    provider,
	}
}

func (p *kubernetesProvider) Spec() *api.DatacenterSpec {
	return &api.DatacenterSpec{
		Description: p.description,
		Country:     p.country,
		Provider:    p.provider,
	}
}

func (p *kubernetesProvider) NewCluster(name string, spec api.ClusterSpec) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return nil, errors.New("not implemented")
}

func (p *kubernetesProvider) Cluster(name string) (*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return nil, errors.New("not implemented")
}

func (p *kubernetesProvider) Clusters() ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	nsList, err := p.client.Namespaces().List(
		labels.SelectorFromSet(labels.Set(map[string]string{"role": "kubermatic-cluster"})),
		fields.Everything(),
	)
	if err != nil {
		return nil, err
	}

	cs := make([]*api.Cluster, 0, len(nsList.Items))
	for i := range nsList.Items {
		ns := nsList.Items[i]
		c := api.Cluster{
			Metadata: api.Metadata{
				Name:     ns.Labels["name"],
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
		}
		cs = append(cs, &c)
	}

	return cs, nil
}

func (p *kubernetesProvider) Nodes(cluster string) ([]string, error) {
	return []string{}, nil
}
