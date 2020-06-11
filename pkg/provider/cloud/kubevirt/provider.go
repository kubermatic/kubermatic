package kubevirt

import (
	"encoding/base64"
	"errors"

	"github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/resources"

	"k8s.io/client-go/tools/clientcmd"
)

type kubevirt struct {
	secretKeySelector provider.SecretKeySelectorValueFunc
}

func NewCloudProvider(secretKeyGetter provider.SecretKeySelectorValueFunc) provider.CloudProvider {
	return &kubevirt{
		secretKeySelector: secretKeyGetter,
	}
}

func (k *kubevirt) DefaultCloudSpec(spec *v1.CloudSpec) error {
	return nil
}

func (k *kubevirt) ValidateCloudSpec(spec v1.CloudSpec) error {
	kubeconfig, err := GetCredentialsForCluster(spec, k.secretKeySelector)
	if err != nil {
		return err
	}

	config, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		// if the decoding failed, the kubeconfig is sent already decoded without the need of decoding it,
		// for example the value has been read from Vault during the ci tests, which is saved as json format.
		config = []byte(kubeconfig)
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

// GetCredentialsForCluster returns the credentials for the passed in cloud spec or an error
func GetCredentialsForCluster(cloud v1.CloudSpec, secretKeySelector provider.SecretKeySelectorValueFunc) (kubeconfig string, err error) {
	kubeconfig = cloud.Kubevirt.Kubeconfig

	if kubeconfig == "" {
		if cloud.Kubevirt.CredentialsReference == nil {
			return "", errors.New("no credentials provided")
		}
		kubeconfig, err = secretKeySelector(cloud.Kubevirt.CredentialsReference, resources.KubevirtKubeConfig)
		if err != nil {
			return "", err
		}
	}

	return kubeconfig, nil
}
