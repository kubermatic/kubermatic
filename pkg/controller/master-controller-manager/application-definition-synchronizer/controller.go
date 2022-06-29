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

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-application-definition-synchronizer"
)

type reconciler struct {
	log          *zap.SugaredLogger
	recorder     record.EventRecorder
	masterClient ctrlruntimeclient.Client
	seedClients  map[string]ctrlruntimeclient.Client
}

func Add(
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	r := &reconciler{
		log:          log.Named(ControllerName),
		recorder:     masterManager.GetEventRecorderFor(ControllerName),
		masterClient: masterManager.GetClient(),
		seedClients:  map[string]ctrlruntimeclient.Client{},
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	c, err := controller.New(ControllerName, masterManager, controller.Options{Reconciler: r, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	// Watch for changes to ApplicationDefinition
	if err := c.Watch(&source.Kind{Type: &appskubermaticv1.ApplicationDefinition{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch for applicationDefinitions: %w", err)
	}

	return nil
}

// Reconcile reconciles ApplicationDefinition objects from master cluster to all seed clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {
	applicationDef := &appskubermaticv1.ApplicationDefinition{}

	if err := r.masterClient.Get(ctx, request.NamespacedName, applicationDef); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

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

	applicationDefCreatorGetters := []reconciling.NamedAppsKubermaticV1ApplicationDefinitionCreatorGetter{
		applicationDefCreatorGetter(applicationDef),
	}

	err := r.syncAllSeeds(log, applicationDef, func(seedClient ctrlruntimeclient.Client, appDef *appskubermaticv1.ApplicationDefinition) error {
		return reconciling.EnsureNamedObjects(ctx, seedClient, "", applicationDefCreatorGetters)
	})
	if err != nil {
		r.recorder.Eventf(applicationDef, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return fmt.Errorf("reconciled application definition: %s: %w", applicationDef.Name, err)
	}

	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, applicationDef *appskubermaticv1.ApplicationDefinition) error {
	if kuberneteshelper.HasFinalizer(applicationDef, appskubermaticv1.ApplicationDefinitionSeedCleanupFinalizer) {
		if err := r.syncAllSeeds(log, applicationDef, func(seedClient ctrlruntimeclient.Client, applicationDef *appskubermaticv1.ApplicationDefinition) error {
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

func (r *reconciler) syncAllSeeds(log *zap.SugaredLogger, applicationDef *appskubermaticv1.ApplicationDefinition, action func(seedClient ctrlruntimeclient.Client, applicationDef *appskubermaticv1.ApplicationDefinition) error) error {
	for seedName, seedClient := range r.seedClients {
		log := log.With("seed", seedName)

		log.Debug("Reconciling application definition with seed")

		err := action(seedClient, applicationDef)
		if err != nil {
			return fmt.Errorf("failed syncing application definition %s for seed %s: %w", applicationDef.Name, seedName, err)
		}
		log.Debug("Reconciled application definition with seed")
	}
	return nil
}

func applicationDefCreatorGetter(applicationDef *appskubermaticv1.ApplicationDefinition) reconciling.NamedAppsKubermaticV1ApplicationDefinitionCreatorGetter {
	return func() (string, reconciling.AppsKubermaticV1ApplicationDefinitionCreator) {
		return applicationDef.Name, func(a *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error) {
			a.Labels = applicationDef.Labels
			a.Annotations = applicationDef.Annotations
			a.Spec = applicationDef.Spec
			return a, nil
		}
	}
}
