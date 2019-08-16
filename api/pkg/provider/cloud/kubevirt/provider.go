package kubevirt

import (
	"encoding/base64"
	"fmt"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"k8s.io/client-go/tools/clientcmd"
)

type kubevirt struct {
}

func NewCloudProvider() provider.CloudProvider {
	return &kubevirt{}
}

func (k *kubevirt) DefaultCloudSpec(spec *v1.CloudSpec) error {
	return nil
}

func (k *kubevirt) ValidateCloudSpec(spec v1.CloudSpec) error {
	if spec.Kubevirt.Config == "" {
		return fmt.Errorf("config cannot be empty")
	}

	config, err := base64.StdEncoding.DecodeString(spec.Kubevirt.Config)
	if err != nil {
		return err
	}

	_, err = clientcmd.RESTConfigFromKubeConfig(config)
	if err != nil {
		return err
	}

	spec.Kubevirt.Config = fmt.Sprintf("%s", config)

	return nil
}

func (k *kubevirt) InitializeCloudProvider(c *v1.Cluster, p provider.ClusterUpdater, s provider.SecretKeySelectorValueFunc) (*v1.Cluster, error) {
	return c, nil
}

func (k *kubevirt) CleanUpCloudProvider(c *v1.Cluster, p provider.ClusterUpdater, s provider.SecretKeySelectorValueFunc) (*v1.Cluster, error) {
	return c, nil
}

func (h *kubevirt) ValidateCloudSpecUpdate(oldSpec v1.CloudSpec, newSpec v1.CloudSpec) error {
	return nil
}
