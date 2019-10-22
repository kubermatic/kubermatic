package usersshkeys

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// CreateUserSSHKeysSecrets creates a secret in the usercluster from the user ssh keys.
func CreateUserSSHKeysSecrets(userSSHKeys map[string][]byte) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.UserSSHKeys, createSecretsFromUserSSHDirPath(userSSHKeys)
	}
}

func createSecretsFromUserSSHDirPath(userSSHKeys map[string][]byte) reconciling.SecretCreator {
	return func(existing *corev1.Secret) (secret *corev1.Secret, e error) {
		existing.Data = map[string][]byte{}

		for k, v := range userSSHKeys {
			existing.Data[k] = v
		}
		return existing, nil
	}
}
