package kubernetes

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
)

var _ provider.KubernetesProvider = (*kubernetesProvider)(nil)

type kubernetesProvider struct {
	mu     sync.Mutex
	cps    map[string]provider.CloudProvider
	client *client.Client

	description, country, provider string
}

// NewKubernetesProvider creates a new kubernetes provider object
func NewKubernetesProvider(
	clientConfig *client.Config,
	cps map[string]provider.CloudProvider,
	desc, country, provider string,
) provider.KubernetesProvider {
	client, err := client.New(clientConfig)
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

	ns, err := p.client.Namespaces().Get(namePrefix + name)
	if err != nil {
		return nil, err
	}

	c, err := unmarshalCluster(p.cps, ns)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (p *kubernetesProvider) Clusters() ([]*api.Cluster, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	nsList, err := p.client.Namespaces().List(
		labels.SelectorFromSet(labels.Set(map[string]string{roleLabelKey: clusterRoleLabel})),
		fields.Everything(),
	)
	if err != nil {
		return nil, err
	}

	cs := make([]*api.Cluster, 0, len(nsList.Items))
	for i := range nsList.Items {
		ns := nsList.Items[i]
		c, err := unmarshalCluster(p.cps, &ns)
		if err != nil {
			log.Println(fmt.Sprintf("error unmarshaling namespace %s: %v", ns.Name, err))
			continue
		}

		cs = append(cs, c)
	}

	return cs, nil
}

func (p *kubernetesProvider) Nodes(cluster string) ([]string, error) {
	return []string{}, nil
}
