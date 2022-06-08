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

package template

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/helmclient"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"

	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// HelmTemplate install upgrade or uninstall helm chart into cluster.
type HelmTemplate struct {
	Ctx context.Context
	// Kubeconfig of the user-cluster.
	Kubeconfig string

	// CacheDir is the directory path where helm caches will be download.
	CacheDir string

	Log                     *zap.SugaredLogger
	ApplicationInstallation *appskubermaticv1.ApplicationInstallation
}

// InstallOrUpgrade the chart located at chartLoc with parameters (releaseName, values) defined applicationInstallation into cluster.
func (h HelmTemplate) InstallOrUpgrade(chartLoc string, applicationInstallation *appskubermaticv1.ApplicationInstallation) error {
	helmCacheDir, err := util.CreateHelmTempDir(h.CacheDir, h.ApplicationInstallation)
	if err != nil {
		return err
	}
	defer util.CleanUpHelmTempDir(helmCacheDir, h.Log)

	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: &h.Kubeconfig,
		Namespace:  &h.ApplicationInstallation.Spec.Namespace.Name,
	}

	helmClient, err := helmclient.NewClient(
		h.Ctx,
		restClientGetter,
		helmclient.NewSettings(helmCacheDir),
		h.ApplicationInstallation.Spec.Namespace.Name,
		h.Log)

	if err != nil {
		return err
	}

	values := make(map[string]interface{})
	if len(applicationInstallation.Spec.Values.Raw) > 0 {
		if err := json.Unmarshal(applicationInstallation.Spec.Values.Raw, &values); err != nil {
			return fmt.Errorf("failed to unmarshall values: %w", err)
		}
	}

	// TODO handle release info
	_, err = helmClient.InstallOrUpgrade(chartLoc, applicationInstallation.Name, values)

	return err
}

// Uninstall the chart from the user cluster.
func (h HelmTemplate) Uninstall(applicationInstallation *appskubermaticv1.ApplicationInstallation) error {
	helmCacheDir, err := util.CreateHelmTempDir(h.CacheDir, h.ApplicationInstallation)
	if err != nil {
		return err
	}
	defer util.CleanUpHelmTempDir(helmCacheDir, h.Log)

	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: &h.Kubeconfig,
		Namespace:  &h.ApplicationInstallation.Spec.Namespace.Name,
	}

	helmClient, err := helmclient.NewClient(
		h.Ctx,
		restClientGetter,
		helmclient.NewSettings(helmCacheDir),
		h.ApplicationInstallation.Spec.Namespace.Name,
		h.Log)

	if err != nil {
		return err
	}

	// TODO handle release info
	_, err = helmClient.Uninstall(applicationInstallation.Name)
	return err
}
