/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/helmclient"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateHelmTempDir creates a temporary directory inside cacheDir where helm caches will be download.
func CreateHelmTempDir(cacheDir string) (string, error) {
	// This will generate a directory like cacheDir-helm-<rand_number> (e.g. /cache/helm-/3012513704)
	downloadDest, err := os.MkdirTemp(cacheDir, "helm-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory where helm caches will be downloaded: %w", err)
	}
	return downloadDest, nil
}

// CleanUpHelmTempDir removes the helm cache directory. If deletion fails error is looged using the logger.
func CleanUpHelmTempDir(cacheDir string, logger *zap.SugaredLogger) {
	if err := os.RemoveAll(cacheDir); err != nil {
		logger.Error("failed to remove temporary directory where helm caches have been downloaded: %s", err)
	}
}

// StatusUpdater is a function that postpone the update of the applicationInstallation.
// It used to set status's filed of a specific template Provider (eg status.HelmRelease).
type StatusUpdater func(status *appskubermaticv1.ApplicationInstallationStatus)

// NoStatusUpdate is a StatusUpdater that does not update the status.
// It useful in case an error happens and we don't have information to update the status.
var NoStatusUpdate StatusUpdater = func(status *appskubermaticv1.ApplicationInstallationStatus) {
	// NO OP
}

// GetCredentialFromSecret get the secret and returns secret.Data[key].
func GetCredentialFromSecret(ctx context.Context, client ctrlruntimeclient.Client, namespce string, name string, key string) (string, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespce, Name: name}, secret); err != nil {
		return "", fmt.Errorf("failed to get credential secret: %w", err)
	}

	cred, found := secret.Data[key]
	if !found {
		return "", fmt.Errorf("key '%s' does not exist in secret '%s'", key, fmt.Sprintf("%s/%s", secret.GetNamespace(), secret.GetName()))
	}
	return string(cred), nil
}

// HelmAuthFromCredentials builds helmclient.AuthSettings from credentials.
// registryConfigFilePath is the path of the file that stores credentials for OCI registry.
// If credentials is nil then an empty helmclient.AuthSettings (i.e. no auth) is returned.
// If credentials can not be extracted from the secret an error is returned.
func HelmAuthFromCredentials(
	ctx context.Context,
	client ctrlruntimeclient.Client,
	registryConfigFilePath string,
	secretNamespace string,
	source *appskubermaticv1.HelmSource,
	credentials *appskubermaticv1.HelmCredentials,
) (helmclient.AuthSettings, error) {
	auth := helmclient.AuthSettings{}
	if source != nil {
		auth = NewAuthSettingsFromHelmSource(source)
	}
	if credentials != nil {
		if credentials.Username != nil {
			username, err := GetCredentialFromSecret(ctx, client, secretNamespace, credentials.Username.Name, credentials.Username.Key)
			if err != nil {
				return auth, err
			}
			auth.Username = username
		}
		if credentials.Password != nil {
			password, err := GetCredentialFromSecret(ctx, client, secretNamespace, credentials.Password.Name, credentials.Password.Key)
			if err != nil {
				return auth, err
			}
			auth.Password = password
		}
		if credentials.RegistryConfigFile != nil {
			registryConfigFile, err := GetCredentialFromSecret(ctx, client, secretNamespace, credentials.RegistryConfigFile.Name, credentials.RegistryConfigFile.Key)
			if err != nil {
				return auth, err
			}
			if err := os.WriteFile(registryConfigFilePath, []byte(registryConfigFile), 0600); err != nil {
				return helmclient.AuthSettings{}, fmt.Errorf("failed to write registryConfigFile: %w", err)
			}
			auth.RegistryConfigFile = registryConfigFilePath
		}
	}
	return auth, nil
}

func NewAuthSettingsFromHelmSource(source *appskubermaticv1.HelmSource) helmclient.AuthSettings {
	auth := helmclient.AuthSettings{}

	if i := source.Insecure; i != nil {
		auth.Insecure = *i
	}

	if p := source.PlainHTTP; p != nil {
		auth.PlainHTTP = *p
	}

	return auth
}
