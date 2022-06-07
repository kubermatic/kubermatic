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

package providers

import (
	"context"
	"errors"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers/source"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SourceProvider is an interface for downloading the application's sources.
type SourceProvider interface {

	// DownloadSource into the destination and return the full path to the source.
	// destination must exist.
	DownloadSource(destination string) (string, error)
}

// NewSourceProvider returns the concrete implementation of SourceProvider according to source defined in appSource.
func NewSourceProvider(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubeconfig string, cacheDir string, appInstallation *appskubermaticv1.ApplicationInstallation, appSource *appskubermaticv1.ApplicationSource, secretNamespace string) (SourceProvider, error) {
	switch {
	case appSource.Helm != nil:
		return source.HelmSource{Ctx: ctx, Kubeconfig: kubeconfig, CacheDir: cacheDir, Log: log, ApplicationInstallation: appInstallation, Source: appSource.Helm}, nil
	case appSource.Git != nil:
		return source.GitSource{Ctx: ctx, Client: client, Source: appSource.Git, SecretNamespace: secretNamespace}, nil
	default: // This should not happen. The admission webhook prevents that.
		return nil, errors.New("no source found")
	}
}
