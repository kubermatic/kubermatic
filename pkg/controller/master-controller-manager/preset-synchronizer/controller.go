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

package presetsynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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
	// This controller syncs the kubermatic preset on the master cluster to the seed clusters.
	ControllerName = "kkp-preset-synchronizer"
)

type reconciler struct {
	log          *zap.SugaredLogger
	masterClient ctrlruntimeclient.Client
	seedClients  map[string]ctrlruntimeclient.Client
	recorder     record.EventRecorder
}

func Add(
	masterMgr manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
) error {
	log = log.Named(ControllerName)
	r := &reconciler{
		log:          log,
		masterClient: masterMgr.GetClient(),
		seedClients:  map[string]ctrlruntimeclient.Client{},
		recorder:     masterMgr.GetEventRecorderFor(ControllerName),
	}

	c, err := controller.New(ControllerName, masterMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	// Watch for changes to Preset
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Preset{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch preset: %w", err)
	}

	return nil
}

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
	preset := &kubermaticv1.Preset{}
	if err := r.masterClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: request.Name}, preset); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	// handling deletion
	if !preset.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, preset); err != nil {
			return fmt.Errorf("handling deletion of preset: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, preset, kubermaticapiv1.PresetSeedCleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	presetCreatorGetters := []reconciling.NamedKubermaticV1PresetCreatorGetter{
		presetCreatorGetter(preset),
	}

	err := r.syncAllSeeds(log, preset, func(seedClient ctrlruntimeclient.Client, preset *kubermaticv1.Preset) error {
		return reconciling.ReconcileKubermaticV1Presets(ctx, presetCreatorGetters, "", seedClient)
	})
	if err != nil {
		r.recorder.Eventf(preset, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return fmt.Errorf("reconciled preset: %s: %w", preset.Name, err)
	}
	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, preset *kubermaticv1.Preset) error {
	if kuberneteshelper.HasFinalizer(preset, kubermaticapiv1.PresetSeedCleanupFinalizer) {
		if err := r.syncAllSeeds(log, preset, func(seedClient ctrlruntimeclient.Client, preset *kubermaticv1.Preset) error {
			err := seedClient.Delete(ctx, &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{
					Name: preset.Name,
				},
			})

			return ctrlruntimeclient.IgnoreNotFound(err)
		}); err != nil {
			return err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, preset, kubermaticapiv1.PresetSeedCleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove preset finalizer %s: %w", preset.Name, err)
		}
	}

	return nil
}

func (r *reconciler) syncAllSeeds(log *zap.SugaredLogger, preset *kubermaticv1.Preset, action func(seedClient ctrlruntimeclient.Client, preset *kubermaticv1.Preset) error) error {
	for seedName, seedClient := range r.seedClients {
		log := log.With("seed", seedName)

		log.Debug("Reconciling preset with seed")

		err := action(seedClient, preset)
		if err != nil {
			return fmt.Errorf("failed syncing preset %s for seed %s: %w", preset.Name, seedName, err)
		}
		log.Debug("Reconciled preset with seed")
	}
	return nil
}

func presetCreatorGetter(preset *kubermaticv1.Preset) reconciling.NamedKubermaticV1PresetCreatorGetter {
	return func() (string, reconciling.KubermaticV1PresetCreator) {
		return preset.Name, func(c *kubermaticv1.Preset) (*kubermaticv1.Preset, error) {
			c.Name = preset.Name
			c.Spec = preset.Spec
			c.Labels = preset.Labels
			return c, nil
		}
	}
}
