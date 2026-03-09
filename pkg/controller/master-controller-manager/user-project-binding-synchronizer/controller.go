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

package userprojectbindingsynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-user-project-binding-synchronizer"

	// cleanupFinalizer indicates that Kubermatic UserProjectBindings on the seed clusters need cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-user-project-bindings"
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

	serviceAccountPredicate := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		// We don't trigger reconciliation for UserProjectBinding of service account.
		userProjectBinding := object.(*kubermaticv1.UserProjectBinding)
		return !kubermaticv1helper.IsProjectServiceAccount(userProjectBinding.Spec.UserEmail)
	})

	_, err := builder.ControllerManagedBy(masterManager).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.UserProjectBinding{}, builder.WithPredicates(serviceAccountPredicate)).
		Watches(&kubermaticv1.Seed{}, enqueueUserProjectBindingsForSeed(r.masterClient, r.log)).
		Build(r)

	return err
}

// Reconcile reconciles Kubermatic UserProjectBinding objects on the master cluster to all seed clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	userProjectBinding := &kubermaticv1.UserProjectBinding{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, userProjectBinding); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !userProjectBinding.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, userProjectBinding); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, userProjectBinding, cleanupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	userProjectBindingReconcilerFactories := []reconciling.NamedUserProjectBindingReconcilerFactory{
		userProjectBindingReconcilerFactory(userProjectBinding),
	}

	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		seedBinding := &kubermaticv1.UserProjectBinding{}
		if err := seedClient.Get(ctx, request.NamespacedName, seedBinding); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch UserProjectBinding on seed cluster: %w", err)
		}

		// see project-synchronizer's syncAllSeeds comment
		if seedBinding.UID != "" && seedBinding.UID == userProjectBinding.UID {
			return nil
		}

		return reconciling.ReconcileUserProjectBindings(ctx, userProjectBindingReconcilerFactories, "", seedClient)
	})

	if err != nil {
		r.recorder.Eventf(userProjectBinding, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconciled userprojectbinding %s: %w", userProjectBinding.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, userProjectBinding *kubermaticv1.UserProjectBinding) error {
	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		return ctrlruntimeclient.IgnoreNotFound(seedClient.Delete(ctx, userProjectBinding))
	})
	if err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, userProjectBinding, cleanupFinalizer)
}

func enqueueUserProjectBindingsForSeed(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		userProjectBindingList := &kubermaticv1.UserProjectBindingList{}
		if err := client.List(ctx, userProjectBindingList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list userprojectbindings: %w", err))
		}
		for _, userProjectBinding := range userProjectBindingList.Items {
			// We don't trigger reconciliation for UserProjectBinding of service account.
			if !kubermaticv1helper.IsProjectServiceAccount(userProjectBinding.Spec.UserEmail) {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Name: userProjectBinding.Name,
				}})
			}
		}
		return requests
	})
}
