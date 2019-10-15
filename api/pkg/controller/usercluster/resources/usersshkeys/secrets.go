package usersshkeys

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

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

		if existing.Data == nil {
			existing.Data = map[string][]byte{}
		}

		if err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {

			}
			fmt.Printf("Path: %v, File name: %v, IsDir: %v", path, info.Name(), info.IsDir())
			return nil
		}); err != nil {
			return nil, fmt.Errorf("failed to walk directory: %v", err)
		}

		for _, file := range files {
			if file.IsDir() {
				continue
			}

			if _, ok := existing.Data[file.Name()]; ok {
				continue
			}

			data, err := ioutil.ReadFile(fmt.Sprintf("%v/%v", path, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read file %v during secret creation: %v", file.Name(), err)
			}
			existing.Data[file.Name()] = data
		}
		return existing, nil
	}
}
