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
	if spec.Kubevirt.Kubeconfig == "" {
		return fmt.Errorf("config cannot be empty")
	}

	config, err := base64.StdEncoding.DecodeString(spec.Kubevirt.Kubeconfig)
	if err != nil {
		// if the decoding failed, the kubeconfig is sent already decoded without the need of decoding it,
		// for example the value has been read from Vault during the ci tests, which is saved as json format.
		config = []byte(spec.Kubevirt.Kubeconfig)
	}

	_, err = clientcmd.RESTConfigFromKubeConfig(config)
	if err != nil {
		return err
	}

	spec.Kubevirt.Kubeconfig = string(config)

	return nil
}

func (k *kubevirt) InitializeCloudProvider(c *v1.Cluster, p provider.ClusterUpdater) (*v1.Cluster, error) {
	return c, nil
}

func (k *kubevirt) CleanUpCloudProvider(c *v1.Cluster, p provider.ClusterUpdater) (*v1.Cluster, error) {
	return c, nil
}

func (k *kubevirt) ValidateCloudSpecUpdate(oldSpec v1.CloudSpec, newSpec v1.CloudSpec) error {
	return nil
}
