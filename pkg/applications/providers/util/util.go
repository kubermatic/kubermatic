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

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"

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
type StatusUpdater func(status *appskubermaticv1.ApplicationInstallationStatus)

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
