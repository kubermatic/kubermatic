package usersshkeys

import (
	"fmt"
	"io/ioutil"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// CreateUserSSHKeysSecrets creates a secret in the usercluster from the user ssh keys.
func CreateUserSSHKeysSecrets(path string) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.UserSSHKeys, createSecretsFromUserSSHDirPath(path)
	}
}

func createSecretsFromUserSSHDirPath(path string) reconciling.SecretCreator {
	return func(existing *corev1.Secret) (secret *corev1.Secret, e error) {
		files, err := ioutil.ReadDir(path)
		if err != nil {
			return nil, err
		}

		existing.Data = map[string][]byte{}
		for _, file := range files {
			data, err := ioutil.ReadFile(fmt.Sprintf("%v/%v", path, file.Name()))
			if err != nil {
				return nil, err
			}
			existing.Data[file.Name()] = data
		}
		return existing, nil
	}
}
