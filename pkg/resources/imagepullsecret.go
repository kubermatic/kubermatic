package resources

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ImagePullSecretCreator returns a creator function to create a ImagePullSecret
func ImagePullSecretCreator(dockerPullConfigJSON []byte) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return ImagePullSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			se.Type = corev1.SecretTypeDockerConfigJson

			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			se.Data[corev1.DockerConfigJsonKey] = dockerPullConfigJSON

			return se, nil
		}
	}
}
