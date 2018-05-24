package kubernetes

import (
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

// NewOptimisticClusterProvider returns a optimistic cluster provider
func NewOptimisticClusterProvider(providers map[string]provider.ClusterProvider, defaultProviderName, workerName string) *OptimisticClusterProvider {
	return &OptimisticClusterProvider{
		providers:           providers,
		workerName:          workerName,
		defaultProviderName: defaultProviderName,
	}
}

// OptimisticClusterProvider is a cluster provider which tries to get the requested cluster from every provider
type OptimisticClusterProvider struct {
	providers map[string]provider.ClusterProvider

	defaultProviderName string
	workerName          string
}

// NewCluster creates a new cluster in the default provider
func (p *OptimisticClusterProvider) NewCluster(user apiv1.User, spec *kubermaticv1.ClusterSpec) (*kubermaticv1.Cluster, error) {
	return p.providers[p.defaultProviderName].NewCluster(user, spec)
}

// Cluster returns the given cluster
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

// Clusters returns all clusters for the given user
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

// DeleteCluster deletes the given cluster
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

// UpdateCluster updates a cluster
func (p *OptimisticClusterProvider) UpdateCluster(user apiv1.User, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	for _, prov := range p.providers {
		_, err := prov.Cluster(user, newCluster.Name)
		if err != nil {
			if err == provider.ErrNotFound {
				continue
			} else {
				return nil, err
			}
		}

		return prov.UpdateCluster(user, newCluster)
	}

	return nil, provider.ErrNotFound
}
