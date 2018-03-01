package kubernetes

import (
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

func NewOptimisticClusterProvider(providers map[string]provider.ClusterProvider, defaultProviderName, workerName string) *OptimisticClusterProvider {
	return &OptimisticClusterProvider{
		providers:           providers,
		workerName:          workerName,
		defaultProviderName: defaultProviderName,
	}
}

type OptimisticClusterProvider struct {
	providers map[string]provider.ClusterProvider

	defaultProviderName string
	workerName          string
}

func (p *OptimisticClusterProvider) NewCluster(user apiv1.User, spec *kubermaticv1.ClusterSpec) (*kubermaticv1.Cluster, error) {
	return p.providers[p.defaultProviderName].NewCluster(user, spec)
}

func (p *OptimisticClusterProvider) Cluster(user apiv1.User, name string) (*kubermaticv1.Cluster, error) {
	for _, prov := range p.providers {
		c, err := prov.Cluster(user, name)
		if err == provider.ErrNotFound {
			continue
		} else {
			return c, err
		}
	}

	return nil, provider.ErrNotFound
}

func (p *OptimisticClusterProvider) Clusters(user apiv1.User) ([]*kubermaticv1.Cluster, error) {
	var clusters []*kubermaticv1.Cluster

	for _, prov := range p.providers {
		c, err := prov.Clusters(user)
		if err != nil {
			return nil, err
		}
		clusters = append(clusters, c...)
	}

	return clusters, nil
}

func (p *OptimisticClusterProvider) DeleteCluster(user apiv1.User, name string) error {
	for _, prov := range p.providers {
		c, err := prov.Cluster(user, name)
		if err != nil {
			if err == provider.ErrNotFound {
				continue
			} else {
				return err
			}
		}

		return prov.DeleteCluster(user, c.Name)
	}

	return provider.ErrNotFound
}

func (p *OptimisticClusterProvider) InitiateClusterUpgrade(user apiv1.User, name, version string) (*kubermaticv1.Cluster, error) {
	for _, prov := range p.providers {
		_, err := prov.Cluster(user, name)
		if err == provider.ErrNotFound {
			continue
		} else {
			return prov.InitiateClusterUpgrade(user, name, version)
		}
	}

	return nil, provider.ErrNotFound
}
