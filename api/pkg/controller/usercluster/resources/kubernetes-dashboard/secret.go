package kubernetesdashboard

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
)

// KeyHolderSecretCreator  TODO(floreks)
func KeyHolderSecretCreator() reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.KubernetesDashboardKeyHolderSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}

			secret.Labels = resources.BaseAppLabel(AppName, nil)
			return secret, nil
		}
	}
}

// CsrfTokenSecretCreator  TODO(floreks)
func CsrfTokenSecretCreator() reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.KubernetesDashboardCsrfTokenSecretName, func(secret *corev1.Secret) (*corev1.Secret, error) {
			if secret.Data == nil {
				secret.Data = map[string][]byte{}
			}

			secret.Labels = resources.BaseAppLabel(AppName, nil)
			return secret, nil
		}
	}
}
