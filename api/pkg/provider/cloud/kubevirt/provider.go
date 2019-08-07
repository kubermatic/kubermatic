package kubevirt

import (
	"fmt"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

type kubevirt struct {
	provider.CloudProvider
}

func NewCloudProvider() provider.CloudProvider {
	return &kubevirt{}
}

func (k *kubevirt) ValidateCloudSpec(spec v1.CloudSpec) error {
	if spec.Kubevirt.Config == "" {
		return fmt.Errorf("namespace cannot be empty")
	}

	return nil
}

func (k *kubevirt) InitializeCloudProvider(c *v1.Cluster, p provider.ClusterUpdater, s provider.SecretKeySelectorValueFunc) (*v1.Cluster, error) {
	return c, nil
}

func (k *kubevirt) CleanUpCloudProvider(c *v1.Cluster,p provider.ClusterUpdater,s provider.SecretKeySelectorValueFunc) (*v1.Cluster, error) {
	return c, nil
}
