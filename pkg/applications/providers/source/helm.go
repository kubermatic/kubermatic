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

package source

import (
	"context"
	"path"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/helmclient"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"

	"k8s.io/cli-runtime/pkg/genericclioptions"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// HelmSource downloads Helm chart from Helm HTTP or OCI registry.
type HelmSource struct {
	Ctx context.Context
	// Kubeconfig of the user-cluster.
	Kubeconfig string
	CacheDir   string
	Log        *zap.SugaredLogger
	Source     *appskubermaticv1.HelmSource
	// Namespace where credential secrets are stored.
	SecretNamespace string

	// SeedClient to seed cluster.
	SeedClient ctrlruntimeclient.Client
}

// DownloadSource downloads the chart into destination folder and return the full path to the chart.
// The destination folder must exist.
func (h HelmSource) DownloadSource(destination string) (string, error) {
	helmCacheDir, err := util.CreateHelmTempDir(h.CacheDir)
	if err != nil {
		return "", err
	}
	defer util.CleanUpHelmTempDir(helmCacheDir, h.Log)

	auth, err := util.HelmAuthFromCredentials(h.Ctx, h.SeedClient, path.Join(helmCacheDir, "reg-creg"), h.SecretNamespace, h.Source, h.Source.Credentials)
	if err != nil {
		return "", err
	}

	// Namespace does not matter to downloading chart.
	ns := "default"
	restClientGetter := &genericclioptions.ConfigFlags{
		KubeConfig: &h.Kubeconfig,
		Namespace:  &ns,
	}

	helmClient, err := helmclient.NewClient(h.Ctx,
		restClientGetter,
		helmclient.NewSettings(helmCacheDir),
		ns,
		h.Log)

	if err != nil {
		return "", err
	}

	return helmClient.DownloadChart(h.Source.URL, h.Source.ChartName, h.Source.ChartVersion, destination, auth)
}
