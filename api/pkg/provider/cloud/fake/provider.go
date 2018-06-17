package fake

import (
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type fakeCloudProvider struct {
	nodes map[string]*apiv1.Node
}

// NewCloudProvider creates a new fake cloud provider
func NewCloudProvider() provider.CloudProvider {
	return &fakeCloudProvider{
		nodes: map[string]*apiv1.Node{},
	}
}

func (p *fakeCloudProvider) ValidateCloudSpec(spec *kubermaticv1.CloudSpec) error {
	return nil
}

func (p *fakeCloudProvider) InitializeCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}

func (p *fakeCloudProvider) CleanUpCloudProvider(cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	return cluster, nil
}
