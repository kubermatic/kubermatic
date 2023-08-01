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

package defaultapplicationinstallationcontroller

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-default-application-installation-controller"
)

// UserClusterClientProvider provides functionality to get a user cluster client.
type UserClusterClientProvider interface {
	GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

type Reconciler struct {
	ctrlruntimeclient.Client

	workerName                    string
	recorder                      record.EventRecorder
	seedGetter                    provider.SeedGetter
	userClusterConnectionProvider UserClusterClientProvider
	log                           *zap.SugaredLogger
	versions                      kubermatic.Versions
}

func Add(mgr manager.Manager, numWorkers int, workerName string, seedGetter provider.SeedGetter, userClusterConnectionProvider UserClusterClientProvider, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		seedGetter:                    seedGetter,
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
		versions:                      versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &kubermaticv1.Cluster{}), &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.DeletionTimestamp != nil {
		// Cluster is queued for deletion; no action required
		r.log.Debugw("Cluster is queued for deletion; no action required", "cluster", cluster.Name)
		return reconcile.Result{}, nil
	}

	// Ensure that cluster is in a state when creating ApplicationInstallation is permissible
	if !cluster.Status.ExtendedHealth.ApplicationControllerHealthy() {
		r.log.Debug("Application controller not healthy")
		return reconcile.Result{}, nil
	}

	var enforcedApplicationList *appskubermaticv1.ApplicationDefinitionList
	err := r.Client.List(ctx, enforcedApplicationList, &ctrlruntimeclient.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("spec.enforce", "true"),
	})
	if err != nil {
		return reconcile.Result{}, err
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	var installedApplicationList *appskubermaticv1.ApplicationInstallationList
	err = userClusterClient.List(ctx, installedApplicationList)
	if err != nil {
		return reconcile.Result{}, err
	}

	var applicationIsInstalled bool
	for _, enforcedApplication := range enforcedApplicationList.Items {
		for _, installedApplication := range installedApplicationList.Items {
			if installedApplication.Spec.ApplicationRef.Name == enforcedApplication.Name {
				applicationIsInstalled = true
				break
			}
		}

		if !applicationIsInstalled {
			applicationInstallation := &appskubermaticv1.ApplicationInstallation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      enforcedApplication.Name,
					Namespace: metav1.NamespaceSystem,
				},
				Spec: appskubermaticv1.ApplicationInstallationSpec{
					Namespace: appskubermaticv1.AppNamespaceSpec{
						Name:   enforcedApplication.Spec.DefaultNamespace,
						Create: true,
					},
					ApplicationRef: appskubermaticv1.ApplicationRef{
						Name:    enforcedApplication.Spec.DefaultName,
						Version: enforcedApplication.Spec.DefaultVersion,
					},
				},
			}

			if enforcedApplication.Spec.DefaultValues != nil {
				applicationInstallation.Spec.Values = *enforcedApplication.Spec.DefaultValues
			}

			err := userClusterClient.Create(ctx, applicationInstallation)
			// If the application already exists, we just ignore the error and move forward.
			if err != nil && ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}
