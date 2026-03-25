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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller syncs the kubermatic preset on the master cluster to the seed clusters.
	ControllerName = "kkp-preset-synchronizer"

	// cleanupFinalizer indicates that synced preset on seed clusters need cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-preset"
)

type reconciler struct {
	log          *zap.SugaredLogger
	masterClient ctrlruntimeclient.Client
	seedClients  kuberneteshelper.SeedClientMap
	recorder     events.EventRecorder
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
		seedClients:  kuberneteshelper.SeedClientMap{},
		recorder:     masterMgr.GetEventRecorder(ControllerName),
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	_, err := builder.ControllerManagedBy(masterMgr).
		Named(ControllerName).
		For(&kubermaticv1.Preset{}).
		Build(r)

	return err
}

func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {
	preset := &kubermaticv1.Preset{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, preset); err != nil {
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	// handling deletion
	if !preset.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, preset); err != nil {
			return fmt.Errorf("handling deletion of preset: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, preset, cleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	presetReconcilerFactories := []reconciling.NamedPresetReconcilerFactory{
		presetReconcilerFactory(preset),
	}

	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		seedPreset := &kubermaticv1.Preset{}
		if err := seedClient.Get(ctx, request.NamespacedName, seedPreset); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch preset on seed cluster: %w", err)
		}

		// see project-synchronizer's syncAllSeeds comment
		if seedPreset.UID != "" && seedPreset.UID == preset.UID {
			return nil
		}

		return reconciling.ReconcilePresets(ctx, presetReconcilerFactories, "", seedClient)
	})
	if err != nil {
		r.recorder.Eventf(preset, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
		return fmt.Errorf("reconciled preset: %s: %w", preset.Name, err)
	}
	return nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, preset *kubermaticv1.Preset) error {
	if kuberneteshelper.HasFinalizer(preset, cleanupFinalizer) {
		if err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
			err := seedClient.Delete(ctx, &kubermaticv1.Preset{
				ObjectMeta: metav1.ObjectMeta{
					Name: preset.Name,
				},
			})

			return ctrlruntimeclient.IgnoreNotFound(err)
		}); err != nil {
			return err
		}

		if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, preset, cleanupFinalizer); err != nil {
			return fmt.Errorf("failed to remove preset finalizer %s: %w", preset.Name, err)
		}
	}

	return nil
}

func presetReconcilerFactory(preset *kubermaticv1.Preset) reconciling.NamedPresetReconcilerFactory {
	return func() (string, reconciling.PresetReconciler) {
		return preset.Name, func(c *kubermaticv1.Preset) (*kubermaticv1.Preset, error) {
			c.Name = preset.Name
			c.Spec = preset.Spec
			c.Labels = preset.Labels
			return c, nil
		}
	}
}
