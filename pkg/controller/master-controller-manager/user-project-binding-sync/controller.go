/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package userprojectbindingsync

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
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
	ControllerName = "user-project-binding-sync-controller"
)

type reconciler struct {
	log              *zap.SugaredLogger
	recorder         record.EventRecorder
	masterClient     ctrlruntimeclient.Client
	seedClientGetter provider.SeedClientGetter
}

func Add(mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	seedKubeconfigGetter provider.SeedKubeconfigGetter) error {

	reconciler := &reconciler{
		log:              log.Named(ControllerName),
		recorder:         mgr.GetEventRecorderFor(ControllerName),
		masterClient:     mgr.GetClient(),
		seedClientGetter: provider.SeedClientGetterFactory(seedKubeconfigGetter),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.UserProjectBinding{}},
		&handler.EnqueueRequestForObject{},
	); err != nil {
		return fmt.Errorf("failed to create watch for userprojectbindings: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Seed{}},
		enqueueAllUserProjectBindings(reconciler.masterClient, reconciler.log),
	); err != nil {
		return fmt.Errorf("failed to create watch for seeds: %v", err)
	}

	return nil
}

// Reconcile reconciles Kubermatic UserProjectBinding objects on the master cluster to all seed clusters
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	userProjectBinding := &kubermaticv1.UserProjectBinding{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, userProjectBinding); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !userProjectBinding.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, userProjectBinding); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %v", err)
		}
		return reconcile.Result{}, nil
	}

	if !kuberneteshelper.HasFinalizer(userProjectBinding, kubermaticapiv1.SeedUserProjectBindingCleanupFinalizer) {
		kuberneteshelper.AddFinalizer(userProjectBinding, kubermaticapiv1.SeedUserProjectBindingCleanupFinalizer)
		if err := r.masterClient.Update(ctx, userProjectBinding); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to add userProjectBinding finalizer %s: %v", userProjectBinding.Name, err)
		}
	}

	userProjectBindingCreatorGetters := []reconciling.NamedKubermaticV1UserProjectBindingCreatorGetter{
		userProjectBindingCreatorGetter(userProjectBinding),
	}

	err := r.syncAllSeeds(ctx, log, userProjectBinding, func(seedClusterClient ctrlruntimeclient.Client, userProjectBinding *kubermaticv1.UserProjectBinding) error {
		return reconciling.ReconcileKubermaticV1UserProjectBindings(ctx, userProjectBindingCreatorGetters, "", seedClusterClient)
	})

	if err != nil {
		r.recorder.Eventf(userProjectBinding, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconciled userprojectbinding: %s: %v", userProjectBinding.Name, err)
	}
	return reconcile.Result{}, nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, userProjectBinding *kubermaticv1.UserProjectBinding) error {
	err := r.syncAllSeeds(ctx, log, userProjectBinding, func(seedClusterClient ctrlruntimeclient.Client, userProjectBinding *kubermaticv1.UserProjectBinding) error {
		if err := seedClusterClient.Delete(ctx, userProjectBinding); err != nil {
			return ctrlruntimeclient.IgnoreNotFound(err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if kuberneteshelper.HasFinalizer(userProjectBinding, kubermaticapiv1.SeedUserProjectBindingCleanupFinalizer) {
		kuberneteshelper.RemoveFinalizer(userProjectBinding, kubermaticapiv1.SeedUserProjectBindingCleanupFinalizer)
		if err := r.masterClient.Update(ctx, userProjectBinding); err != nil {
			return fmt.Errorf("failed to remove userprojectbinding finalizer %s: %v", userProjectBinding.Name, err)
		}
	}
	return nil
}

func (r *reconciler) syncAllSeeds(
	ctx context.Context,
	log *zap.SugaredLogger,
	userProjectBinding *kubermaticv1.UserProjectBinding,
	action func(seedClusterClient ctrlruntimeclient.Client, userProjectBinding *kubermaticv1.UserProjectBinding) error) error {

	seedList := &kubermaticv1.SeedList{}
	if err := r.masterClient.List(ctx, seedList); err != nil {
		return fmt.Errorf("failed listing seeds: %w", err)
	}

	for _, seed := range seedList.Items {
		seedClient, err := r.seedClientGetter(&seed)
		if err != nil {
			return fmt.Errorf("failed getting seed client for seed %s: %w", seed.Name, err)
		}

		err = action(seedClient, userProjectBinding)
		if err != nil {
			return fmt.Errorf("failed syncing userprojectbinding for seed %s: %w", seed.Name, err)
		}
		log.Debugw("Reconciled userprojectbinding with seed", "seed", seed.Name)
	}
	return nil
}

func enqueueAllUserProjectBindings(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		userProjectBindingList := &kubermaticv1.UserProjectBindingList{}
		if err := client.List(context.Background(), userProjectBindingList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list userprojectbindings: %v", err))
		}
		for _, userProjectBinding := range userProjectBindingList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name: userProjectBinding.Name,
			}})
		}
		return requests
	})
}
