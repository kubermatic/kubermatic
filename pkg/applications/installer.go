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
	"os"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/applications/providers"
	"k8c.io/kubermatic/v2/pkg/applications/providers/util"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplicationInstaller handles the installation / uninstallation of an Application on the user cluster.
type ApplicationInstaller interface {
	// Apply function installs the application on the user-cluster and returns an error if the installation has failed; this is idempotent.
	Apply(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error)

	// Delete function uninstalls the application on the user-cluster and returns an error if the uninstallation has failed; this is idempotent.
	Delete(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error)
}

// ApplicationManager handles the installation / uninstallation of an Application on the user-cluster.
type ApplicationManager struct {
	// ApplicationCache is the path to the directory used for caching applications. (i.e. location where application's source will be downloaded, Helm repository cache ...)
	ApplicationCache string

	// Kubeconfig of the user-cluster
	Kubeconfig string

	// Namespace where credentials secrets are stored.
	SecretNamespace string
}

// Apply creates the namespace where the application will be installed (if necessary) and installs the application.
func (a *ApplicationManager) Apply(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error) {
	// initialize tools
	sourceProvider, err := providers.NewSourceProvider(ctx, log, seedClient, a.Kubeconfig, a.ApplicationCache, applicationInstallation, &applicationInstallation.Status.ApplicationVersion.Template.Source, a.SecretNamespace)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize source provider: %w", err)
	}

	downloadDest, err := os.MkdirTemp(a.ApplicationCache, applicationInstallation.Namespace+"-"+applicationInstallation.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary directory where application source will be downloaded: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(downloadDest); err != nil {
			log.Error("failed to remove temporary directory where application source has been downloaded: %s", err)
		}
	}()

	templateProvider, err := providers.NewTemplateProvider(ctx, a.Kubeconfig, a.ApplicationCache, log, applicationInstallation, applicationInstallation.Status.ApplicationVersion.Template.Method)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize template provider: %w", err)
	}

	// start reconciliation
	if err := a.reconcileNamespace(ctx, log, applicationInstallation, userClient); err != nil {
		return nil, err
	}

	appSourcePath, err := sourceProvider.DownloadSource(downloadDest)
	if err != nil {
		return nil, fmt.Errorf("failed to download application source: %w", err)
	}

	return templateProvider.InstallOrUpgrade(appSourcePath, applicationInstallation)
}

// Delete uninstalls the application and deletes the namespace where the application was installed if necessary.
func (a *ApplicationManager) Delete(ctx context.Context, log *zap.SugaredLogger, seedClient ctrlruntimeclient.Client, userClient ctrlruntimeclient.Client, applicationInstallation *appskubermaticv1.ApplicationInstallation) (util.StatusUpdater, error) {
	templateProvider, err := providers.NewTemplateProvider(ctx, a.Kubeconfig, a.ApplicationCache, log, applicationInstallation, applicationInstallation.Status.ApplicationVersion.Template.Method)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize template provider: %w", err)
	}

	statuUpdater, err := templateProvider.Uninstall(applicationInstallation)
	if err != nil {
		return nil, err
	}

	return statuUpdater, a.deleteNamespace(ctx, log, applicationInstallation, userClient)
}

// reconcileNamespace ensures namespace is created and has desired labels and annotations if applicationInstallation.Spec.Namespace.Create flag is set.
func (a *ApplicationManager) reconcileNamespace(ctx context.Context, log *zap.SugaredLogger, applicationInstallation *appskubermaticv1.ApplicationInstallation, userClient ctrlruntimeclient.Client) error {
	desiredNs := applicationInstallation.Spec.Namespace
	if desiredNs.Create {
		log.Infof("reconciling namespace '%s'", desiredNs.Name)

		creators := []reconciling.NamedNamespaceCreatorGetter{
			func() (name string, create reconciling.NamespaceCreator) {
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

// deleteNamespace delete the namespace if applicationInstallation.Spec.Namespace.Create flag is set.
func (a *ApplicationManager) deleteNamespace(ctx context.Context, log *zap.SugaredLogger, applicationInstallation *appskubermaticv1.ApplicationInstallation, userClient ctrlruntimeclient.Client) error {
	desiredNs := applicationInstallation.Spec.Namespace
	if desiredNs.Create {
		log.Infof("deleting namespace '%s'", desiredNs.Name)
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: desiredNs.Name,
			},
		}

		if err := userClient.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete namespace: %w", err)
		}
	}

	return nil
}
