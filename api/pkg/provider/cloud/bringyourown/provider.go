package bringyourown

import (
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type bringyourown struct{}

// NewCloudProvider creates a new bringyourown provider.
func NewCloudProvider() provider.CloudProvider {
	return &bringyourown{}
}

func (b *bringyourown) ValidateCloudSpec(*kubermaticv1.CloudSpec) error {
	return nil
}

func (b *bringyourown) InitializeCloudProvider(cloud *kubermaticv1.CloudSpec, name string) (*kubermaticv1.CloudSpec, error) {
	return nil, nil
}

func (b *bringyourown) CleanUpCloudProvider(*kubermaticv1.CloudSpec) error {
	return nil
}

func (b *bringyourown) ValidateNodeSpec(cloudSpec *kubermaticv1.CloudSpec, nodeSpec *apiv1.NodeSpec) error {
	return nil
}
