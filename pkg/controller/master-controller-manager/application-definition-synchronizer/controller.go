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

package applicationdefinitionsynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-application-definition-synchronizer"
)

type reconciler struct {
	log          *zap.SugaredLogger
	recorder     events.EventRecorder
	masterClient ctrlruntimeclient.Client
	seedClients  kuberneteshelper.SeedClientMap
}

func Add(
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	r := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     masterManager.GetEventRecorder(ControllerName),
		masterClient: masterManager.GetClient(),
		seedClients:  kuberneteshelper.SeedClientMap{},
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	_, err := builder.ControllerManagedBy(masterManager).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&appskubermaticv1.ApplicationDefinition{}).
		Build(r)

	return err
}

// Reconcile reconciles ApplicationDefinition objects from master cluster to all seed clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("appdefinition", request.Name)
	log.Debug("Processing")

	applicationDef := &appskubermaticv1.ApplicationDefinition{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, applicationDef); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, log, applicationDef)
	if err != nil {
		r.recorder.Eventf(applicationDef, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, applicationDef *appskubermaticv1.ApplicationDefinition) error {
	// handling deletion
	if !applicationDef.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, applicationDef); err != nil {
			return fmt.Errorf("handling deletion of application definition: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, applicationDef, appskubermaticv1.ApplicationDefinitionSeedCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	applicationDefReconcilerFactories := []reconciling.NamedApplicationDefinitionReconcilerFactory{
		applicationDefReconcilerFactory(applicationDef),
	}

	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		log.Debug("Reconciling application definition with seed")

		seedDef := &appskubermaticv1.ApplicationDefinition{}
		if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(applicationDef), seedDef); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch ApplicationDefinition on seed cluster: %w", err)
		}

		// see project-synchronizer's syncAllSeeds comment
		if seedDef.UID != "" && seedDef.UID == applicationDef.UID {
			return nil
		}

		return reconciling.ReconcileApplicationDefinitions(ctx, applicationDefReconcilerFactories, "", seedClient)
	})
	if err != nil {
		return fmt.Errorf("reconciled application definition %s: %w", applicationDef.Name, err)
	}

	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, applicationDef *appskubermaticv1.ApplicationDefinition) error {
	if kuberneteshelper.HasFinalizer(applicationDef, appskubermaticv1.ApplicationDefinitionSeedCleanupFinalizer) {
		if err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
			log.Debug("Deleting application definition on seed")

			err := seedClient.Delete(ctx, &appskubermaticv1.ApplicationDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: applicationDef.Name,
				},
			})

			return ctrlruntimeclient.IgnoreNotFound(err)
		}); err != nil {
			return err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, applicationDef, appskubermaticv1.ApplicationDefinitionSeedCleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove application definition finalizer %s: %w", applicationDef.Name, err)
		}
	}
	return nil
}

func applicationDefReconcilerFactory(applicationDef *appskubermaticv1.ApplicationDefinition) reconciling.NamedApplicationDefinitionReconcilerFactory {
	return func() (string, reconciling.ApplicationDefinitionReconciler) {
		return applicationDef.Name, func(a *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error) {
			a.Labels = applicationDef.Labels
			a.Annotations = applicationDef.Annotations
			a.Spec = applicationDef.Spec
			return a, nil
		}
	}
}
