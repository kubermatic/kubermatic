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

package groupprojectbindingsynchronizer

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "group-project-binding-sync-controller"
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

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.GroupProjectBinding{}},
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return fmt.Errorf("failed to create watch for groupprojectbindings: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Seed{}},
		enqueueGroupProjectBindingsForSeed(r.masterClient, r.log),
	); err != nil {
		return fmt.Errorf("failed to create watch for seeds: %w", err)
	}

	return nil
}

// Reconcile reconciles Kubermatic GroupProjectBinding objects on the master cluster to all seed clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	groupProjectBinding := &kubermaticv1.GroupProjectBinding{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, groupProjectBinding); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !groupProjectBinding.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, groupProjectBinding); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, groupProjectBinding, apiv1.SeedGroupProjectBindingCleanupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	groupProjectBindingCreatorGetters := []reconciling.NamedKubermaticV1GroupProjectBindingCreatorGetter{
		groupProjectBindingCreatorGetter(groupProjectBinding),
	}

	err := r.syncAllSeeds(log, groupProjectBinding, func(seedClusterClient ctrlruntimeclient.Client, groupProjectBinding *kubermaticv1.GroupProjectBinding) error {
		return reconciling.ReconcileKubermaticV1GroupProjectBindings(ctx, groupProjectBindingCreatorGetters, "", seedClusterClient)
	})

	if err != nil {
		r.recorder.Eventf(groupProjectBinding, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, fmt.Errorf("failed to reconcile groupprojectbinding '%s': %w", groupProjectBinding.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, groupProjectBinding *kubermaticv1.GroupProjectBinding) error {
	err := r.syncAllSeeds(log, groupProjectBinding,
		func(seedClusterClient ctrlruntimeclient.Client, groupProjectBinding *kubermaticv1.GroupProjectBinding) error {
			if err := seedClusterClient.Delete(ctx, groupProjectBinding); err != nil {
				return ctrlruntimeclient.IgnoreNotFound(err)
			}

			return nil
		},
	)

	if err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, groupProjectBinding, apiv1.SeedGroupProjectBindingCleanupFinalizer)
}

type actionFunc func(seedClusterClient ctrlruntimeclient.Client, groupProjectBinding *kubermaticv1.GroupProjectBinding) error

func (r *reconciler) syncAllSeeds(log *zap.SugaredLogger, groupProjectBinding *kubermaticv1.GroupProjectBinding, action actionFunc) error {
	seedErrs := []error{}

	for seedName, seedClient := range r.seedClients {
		if err := action(seedClient, groupProjectBinding); err != nil {
			log.Errorf("failed to sync GroupProjectBinding for seed '%s': %w", seedName, err)
			seedErrs = append(seedErrs, err)
		}

		log.Debugw("reconciled groupprojectbinding with seed", "seed", seedName)
	}

	if len(seedErrs) > 0 {
		slice := []string{}
		for _, err := range seedErrs {
			slice = append(slice, err.Error())
		}

		return fmt.Errorf("failed to sync to at least one seed: %s", strings.Join(slice, ","))
	}

	return nil
}

func enqueueGroupProjectBindingsForSeed(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		groupProjectBindingList := &kubermaticv1.GroupProjectBindingList{}

		if err := client.List(context.Background(), groupProjectBindingList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list userprojectbindings: %w", err))
		}

		for _, groupProjectBinding := range groupProjectBindingList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name: groupProjectBinding.Name,
			}})
		}

		return requests
	})
}
