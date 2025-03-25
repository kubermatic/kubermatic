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

package applications

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplicationInstaller handles the installation / uninstallation of an Application on the user cluster.
type ApplicationInstaller interface {
	// GetAppCache return the application cache location (i.e. where source and others temporary files are written)
	GetAppCache() string

	// DownloadSource the application's source into downloadDest and returns the full path to the sources.
	DownloadSource(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation, downloadDest string) (string, error)

	// Apply function installs the application on the user-cluster and returns an error if the installation has failed. StatusUpdater is guaranteed to be non nil. This is idempotent.
	Apply(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation, appSourcePath string) (util.StatusUpdater, error)

	// Delete function uninstalls the application on the user-cluster and returns an error if the uninstallation has failed. StatusUpdater is guaranteed to be non nil. This is idempotent.
	Delete(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error)

	// IsStuck determines if a release is stuck. Its main purpose is to detect inconsistent behavior in upstream Application libraries
	IsStuck(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error)

	// Rollback rolls an Application back to the previous release
	Rollback(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) error
}

// ApplicationManager handles the installation / uninstallation of an Application on the user-cluster.
type ApplicationManager struct {
	// ApplicationCache is the path to the directory used for caching applications. (i.e. location where application's source will be downloaded, Helm repository cache ...)
	ApplicationCache string

	// Kubeconfig of the user-cluster
	Kubeconfig string

	// Namespace where credentials secrets are stored.
	SecretNamespace string

	// ClusterName of the user-cluster
	ClusterName string
}

func (a *ApplicationManager) GetAppCache() string {
	return a.ApplicationCache
}

// DownloadSource the application's source using the appropriate provider into downloadDest and returns the full path to the sources.
func (a *ApplicationManager) DownloadSource(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation, downloadDest string) (string, error) {
	sourceProvider, err := providers.NewSourceProvider(ctx, log, seedClient, a.Kubeconfig, a.ApplicationCache, &applicationInstallation.Status.ApplicationVersion.Template.Source, a.SecretNamespace)
	if err != nil {
		return "", fmt.Errorf("failed to initialize source provider: %w", err)
	}

	appSourcePath, err := sourceProvider.DownloadSource(downloadDest)
	if err != nil {
		return "", fmt.Errorf("failed to download application source: %w", err)
	}

	return appSourcePath, nil
}

// Apply creates the namespace where the application will be installed (if necessary) and installs the application.
func (a *ApplicationManager) Apply(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, appDefinition *appskubermaticv1.ApplicationDefinition, applicationInstallation *appskubermaticv1.ApplicationInstallation, appSourcePath string) (util.StatusUpdater, error) {
	templateProvider, err := providers.NewTemplateProvider(ctx, seedClient, a.ClusterName, a.Kubeconfig, a.ApplicationCache, log, applicationInstallation, a.SecretNamespace)
	if err != nil {
		return util.NoStatusUpdate, fmt.Errorf("failed to initialize template provider: %w", err)
	}

	// start reconciliation
	if err := a.reconcileNamespace(ctx, log, applicationInstallation, userClient); err != nil {
		return util.NoStatusUpdate, err
	}
	return templateProvider.InstallOrUpgrade(appSourcePath, appDefinition, applicationInstallation)
}

// Delete uninstalls the application where the application was installed if necessary.
func (a *ApplicationManager) Delete(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error) {
	templateProvider, err := providers.NewTemplateProvider(ctx, seedClient, a.ClusterName, a.Kubeconfig, a.ApplicationCache, log, applicationInstallation, a.SecretNamespace)
	if err != nil {
		return util.NoStatusUpdate, fmt.Errorf("failed to initialize template provider: %w", err)
	}

	return templateProvider.Uninstall(applicationInstallation)
}

// reconcileNamespace ensures namespace is created and has desired labels and annotations if applicationInstallation.Spec.Namespace.Create flag is set.
func (a *ApplicationManager) reconcileNamespace(ctx context.Context, log *zap.SugaredLogger, applicationInstallation *appskubermaticv1.ApplicationInstallation, userClient ctrlruntimeclient.Client) error {
	desiredNs := applicationInstallation.Spec.Namespace
	if desiredNs.Create {
		log.Infow("reconciling namespace", "namespace", desiredNs.Name)

		creators := []reconciling.NamedNamespaceReconcilerFactory{
			func() (name string, reconciler reconciling.NamespaceReconciler) {
				return desiredNs.Name, func(ns *corev1.Namespace) (*corev1.Namespace, error) {
					if desiredNs.Labels != nil {
						if ns.Labels == nil {
							ns.Labels = map[string]string{}
						}
						for k, v := range desiredNs.Labels {
							ns.Labels[k] = v
						}
					}

					if desiredNs.Annotations != nil {
						if ns.Annotations == nil {
							ns.Annotations = map[string]string{}
						}
						for k, v := range desiredNs.Annotations {
							ns.Annotations[k] = v
						}
					}
					return ns, nil
				}
			},
		}

		if err := reconciling.ReconcileNamespaces(ctx, creators, "", userClient); err != nil {
			return fmt.Errorf("failed to reconcile namespace: %w", err)
		}
	}
	return nil
}

// IsStuck determines if a release is stuck. Its main purpose is to detect inconsistent behavior in upstream Application libraries.
func (a *ApplicationManager) IsStuck(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (bool, error) {
	templateProvider, err := providers.NewTemplateProvider(ctx, seedClient, a.ClusterName, a.Kubeconfig, a.ApplicationCache, log, applicationInstallation, a.SecretNamespace)
	if err != nil {
		return false, fmt.Errorf("failed to initialize template provider: %w", err)
	}

	return templateProvider.IsStuck(applicationInstallation)
}

// Rollback rolls an Application back to the previous release.
func (a *ApplicationManager) Rollback(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) error {
	templateProvider, err := providers.NewTemplateProvider(ctx, seedClient, a.ClusterName, a.Kubeconfig, a.ApplicationCache, log, applicationInstallation, a.SecretNamespace)
	if err != nil {
		return fmt.Errorf("failed to initialize template provider: %w", err)
	}

	return templateProvider.Rollback(applicationInstallation)
}
