/*
 Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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
	"fmt"
	"os"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
)

// CreateHelmTempDir creates a temporary directory inside cacheDir where helm caches will be download.
func CreateHelmTempDir(cacheDir string, applicationInstallation *appskubermaticv1.ApplicationInstallation) (string, error) {
	downloadDest, err := os.MkdirTemp(cacheDir, "helm-"+applicationInstallation.Namespace+"-"+applicationInstallation.Name+"-")
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
