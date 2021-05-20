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

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "user-synchronizer"
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

	c, err := controller.New(ControllerName, masterManager, controller.Options{Reconciler: r, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	serviceAccountPredicate := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		// We don't trigger reconciliation for service account.
		return !kubernetes.IsProjectServiceAccount(object.GetName())
	})

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
		seedClusterWatch := &source.Kind{Type: &kubermaticv1.User{}}
		if err := seedClusterWatch.InjectCache(seedManager.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache for seed %q in to watch: %w", seedName, err)
		}
		if err := c.Watch(seedClusterWatch, &handler.EnqueueRequestForObject{}, serviceAccountPredicate); err != nil {
			return fmt.Errorf("failed to watch user objects in seed %q: %w", seedName, err)
		}
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.User{}}, &handler.EnqueueRequestForObject{}, serviceAccountPredicate,
	); err != nil {
		return fmt.Errorf("failed to create watch for user objects in master cluster: %w", err)
	}

	return nil
}

// Reconcile reconciles Kubermatic User objects (excluding service account users) on the master cluster to all seed clusters
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)

	user := &kubermaticv1.User{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, user); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	if !user.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, user); err != nil {
			return reconcile.Result{}, fmt.Errorf("handling deletion: %w", err)
		}
		return reconcile.Result{}, nil
	}

	if !kuberneteshelper.HasFinalizer(user, kubermaticapiv1.SeedUserCleanupFinalizer) {
		kuberneteshelper.AddFinalizer(user, kubermaticapiv1.SeedUserCleanupFinalizer)
		if err := r.masterClient.Update(ctx, user); err != nil {
			return reconcile.Result{}, fmt.Errorf("failed to add user finalizer %s: %w", user.Name, err)
		}
	}

	userCreatorGetters := []reconciling.NamedKubermaticV1UserCreatorGetter{
		userCreatorGetter(user),
	}
	err := r.syncAllSeeds(log, user, func(seedClusterClient ctrlruntimeclient.Client, user *kubermaticv1.User) error {
		return reconciling.ReconcileKubermaticV1Users(ctx, userCreatorGetters, "", seedClusterClient)
	})
	if err != nil {
		r.recorder.Eventf(user, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconciled user: %s: %w", user.Name, err)
	}
	return reconcile.Result{}, nil
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, user *kubermaticv1.User) error {
	err := r.syncAllSeeds(log, user, func(seedClusterClient ctrlruntimeclient.Client, user *kubermaticv1.User) error {
		if err := seedClusterClient.Delete(ctx, user); err != nil {
			return ctrlruntimeclient.IgnoreNotFound(err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if kuberneteshelper.HasFinalizer(user, kubermaticapiv1.SeedUserCleanupFinalizer) {
		kuberneteshelper.RemoveFinalizer(user, kubermaticapiv1.SeedUserCleanupFinalizer)
		if err := r.masterClient.Update(ctx, user); err != nil {
			return fmt.Errorf("failed to remove user finalizer %s: %w", user.Name, err)
		}
	}
	return nil
}

func (r *reconciler) syncAllSeeds(
	log *zap.SugaredLogger,
	user *kubermaticv1.User,
	action func(seedClusterClient ctrlruntimeclient.Client, user *kubermaticv1.User) error) error {
	for seedName, seedClient := range r.seedClients {
		err := action(seedClient, user)
		if err != nil {
			return fmt.Errorf("failed syncing user for seed %s: %w", seedName, err)
		}
		log.Debugw("Reconciled user with seed", "seed", seedName)
	}
	return nil
}
