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
	"fmt"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers/source"
	"k8c.io/kubermatic/v2/pkg/applications/providers/template"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// SourceProvider is an interface for downloading the application's sources.
type SourceProvider interface {

	// DownloadSource into the destination and return the full path to the source.
	// destination must exist.
	DownloadSource(destination string) (string, error)
}

// NewSourceProvider returns the concrete implementation of SourceProvider according to source defined in appSource.
func NewSourceProvider(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, kubeconfig string, cacheDir string, appSource *appskubermaticv1.ApplicationSource, secretNamespace string) (SourceProvider, error) {
	switch {
	case appSource.Helm != nil:
		return source.HelmSource{Ctx: ctx, SeedClient: client, Kubeconfig: kubeconfig, CacheDir: cacheDir, Log: log, Source: appSource.Helm, SecretNamespace: secretNamespace}, nil
	case appSource.Git != nil:
		return source.GitSource{Ctx: ctx, SeedClient: client, Source: appSource.Git, SecretNamespace: secretNamespace}, nil
	default: // This should not happen. The admission webhook prevents that.
		return nil, errors.New("no source found")
	}
}

// TemplateProvider is an interface to install, upgrade or uninstall application.
type TemplateProvider interface {

	// InstallOrUpgrade the application from the source.
	InstallOrUpgrade(source string, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error)

	// Uninstall the application.
	Uninstall(applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error)

	// IsStuck checks if a release is stuck
	IsStuck(applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error)

	// Rollback the Application to the previous release
	Rollback(applicationInstallation *appskubermaticv1.ApplicationInstallation) error
}

// NewTemplateProvider return the concrete implementation of TemplateProvider according to the templateMethod.
func NewTemplateProvider(ctx context.Context, seedClient ctrlruntimeclient.Client, clusterName string, kubeconfig string, cacheDir string, log *zap.SugaredLogger, appInstallation *appskubermaticv1.ApplicationInstallation, secretNamespace string) (TemplateProvider, error) {
	switch appInstallation.Status.Method {
	case appskubermaticv1.HelmTemplateMethod:
		return template.HelmTemplate{Ctx: ctx, Kubeconfig: kubeconfig, CacheDir: cacheDir, Log: log, SecretNamespace: secretNamespace, ClusterName: clusterName, SeedClient: seedClient}, nil
	default:
		return nil, fmt.Errorf("template method '%v' not implemented", appInstallation.Status.Method)
	}
}
