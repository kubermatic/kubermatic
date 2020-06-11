package cloudcontroller

import (
	"github.com/kubermatic/kubermatic/pkg/resources"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"
	corev1 "k8s.io/api/core/v1"
)

// CloudConfig generates the cloud-config secret to be injected in the user cluster.
func CloudConfig(cloudConfig []byte) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.CloudConfigSecretName, func(existing *corev1.Secret) (*corev1.Secret, error) {
			existing.Name = resources.CloudConfigSecretName
			if existing.Data == nil {
				existing.Data = map[string][]byte{}
			}
			existing.Data[resources.CloudConfigSecretKey] = cloudConfig
			return existing, nil
		}

	}
}
