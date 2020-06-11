package usersshkeys

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// SecretCreator returns a function to create a secret in the usercluster containing the user ssh keys.
func SecretCreator(userSSHKeys map[string][]byte) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.UserSSHKeys, func(sec *corev1.Secret) (*corev1.Secret, error) {
			sec.Data = map[string][]byte{}
			for k, v := range userSSHKeys {
				sec.Data[k] = v
			}

			return sec, nil
		}
	}
}
