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

package usersynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"k8c.io/kubermatic/sdk/v2/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-user-synchronizer"

	// cleanupFinalizer indicates that Kubermatic Users on the seed clusters need cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-users"
)

type reconciler struct {
	log             *zap.SugaredLogger
	recorder        events.EventRecorder
	masterClient    ctrlruntimeclient.Client
	masterAPIReader ctrlruntimeclient.Reader
	seedClients     kuberneteshelper.SeedClientMap
}

func Add(
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
) error {
	r := &reconciler{
		log:             log.Named(ControllerName),
		recorder:        masterManager.GetEventRecorder(ControllerName),
		masterClient:    masterManager.GetClient(),
		masterAPIReader: masterManager.GetAPIReader(),
		seedClients:     kuberneteshelper.SeedClientMap{},
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	serviceAccountPredicate := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		// We don't trigger reconciliation for service account.
		user := object.(*kubermaticv1.User)
		return !kubermaticv1helper.IsProjectServiceAccount(user.Spec.Email)
	})

	_, err := builder.ControllerManagedBy(masterManager).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.User{}, builder.WithPredicates(serviceAccountPredicate, withEventFilter())).
		Build(r)

	return err
}

func withEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return true
		},
	}
}

// Reconcile reconciles Kubermatic User objects (excluding service account users) on the master cluster to all seed clusters.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	user := &kubermaticv1.User{}
	// using the reader here to bypass the cache. It is necessary because we update the same object we are watching
	// in the case when master and seed clusters are on the same cluster. Otherwise, the old cache state can overwrite
	// the update. Ideally, we would not reconcile the resource whose change caused the reconciliation.
	if err := r.masterAPIReader.Get(ctx, request.NamespacedName, user); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !user.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, user); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, user, cleanupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	userReconcilerFactories := []reconciling.NamedUserReconcilerFactory{
		userReconcilerFactory(user),
	}
	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		seedUser := &kubermaticv1.User{}
		if err := seedClient.Get(ctx, request.NamespacedName, seedUser); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch user on seed cluster: %w", err)
		}

		// see project-synchronizer's syncAllSeeds comment
		if seedUser.UID != "" && seedUser.UID == user.UID {
			return nil
		}

		err := reconciling.ReconcileUsers(ctx, userReconcilerFactories, "", seedClient)
		if err != nil {
			return fmt.Errorf("failed to reconcile user: %w", err)
		}

		if err := seedClient.Get(ctx, request.NamespacedName, seedUser); err != nil {
			return fmt.Errorf("failed to fetch user on seed cluster: %w", err)
		}

		if !equality.Semantic.DeepEqual(user.Status, seedUser.Status) {
			oldSeedUser := seedUser.DeepCopy()
			seedUser.Status = *user.Status.DeepCopy()
			if err := seedClient.Status().Patch(ctx, seedUser, ctrlruntimeclient.MergeFrom(oldSeedUser)); err != nil {
				return fmt.Errorf("failed to update user status on seed cluster: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		r.recorder.Eventf(user, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconciled user %s: %w", user.Name, err)
	}

	return reconcile.Result{}, nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, user *kubermaticv1.User) error {
	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		return ctrlruntimeclient.IgnoreNotFound(seedClient.Delete(ctx, user))
	})
	if err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, user, cleanupFinalizer)
}
