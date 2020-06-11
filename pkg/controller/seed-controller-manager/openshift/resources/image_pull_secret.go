package resources

import (
	"errors"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const openshiftImagePullSecretName = "openshift-image-pull-secret"

func ImagePullSecretCreator(cluster *kubermaticv1.Cluster) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return openshiftImagePullSecretName, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.Type = corev1.SecretTypeDockerConfigJson
			if s.Data == nil {
				s.Data = map[string][]byte{}
			}
			// Should never happen
			if cluster.Spec.Openshift == nil {
				return nil, errors.New("openshift spec is nil")
			}
			s.Data[corev1.DockerConfigJsonKey] = []byte(cluster.Spec.Openshift.ImagePullSecret)
			return s, nil
		}
	}
}
