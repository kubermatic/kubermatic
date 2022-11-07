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
	"strings"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/apis/equality"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-user-synchronizer"

	// seedUsersCleanupFinalizer indicates that Kubermatic Users on the seed clusters need cleanup.
	seedUsersCleanupFinalizer = "kubermatic.k8c.io/cleanup-seed-users"

	// masterUserProjectBindingCleanupFinalizer indicates that Kubermatic UserProjectBindings on the master cluster need cleanup.
	masterUserProjectBindingCleanupFinalizer = "kubermatic.io/cleanup-master-user-project-bindings"
)

type reconciler struct {
	log             *zap.SugaredLogger
	recorder        record.EventRecorder
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
		recorder:        masterManager.GetEventRecorderFor(ControllerName),
		masterClient:    masterManager.GetClient(),
		masterAPIReader: masterManager.GetAPIReader(),
		seedClients:     kuberneteshelper.SeedClientMap{},
	}

	c, err := controller.New(ControllerName, masterManager, controller.Options{Reconciler: r, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	serviceAccountPredicate := predicate.NewPredicateFuncs(func(object ctrlruntimeclient.Object) bool {
		// We don't trigger reconciliation for service account.
		user := object.(*kubermaticv1.User)
		return !kubermaticv1helper.IsProjectServiceAccount(user.Spec.Email)
	})

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.User{}}, &handler.EnqueueRequestForObject{}, serviceAccountPredicate, withEventFilter(),
	); err != nil {
		return fmt.Errorf("failed to create watch for user objects in master cluster: %w", err)
	}

	return nil
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

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, user, masterUserProjectBindingCleanupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer %v to user %s: %w", masterUserProjectBindingCleanupFinalizer, user.Name, err)
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, user, seedUsersCleanupFinalizer); err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
	}

	userCreatorGetters := []reconciling.NamedKubermaticV1UserCreatorGetter{
		userCreatorGetter(user),
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

		err := reconciling.ReconcileKubermaticV1Users(ctx, userCreatorGetters, "", seedClient)
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
		r.recorder.Event(user, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		return reconcile.Result{}, fmt.Errorf("reconciled user: %s: %w", user.Name, err)
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
	if err := kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, user, seedUsersCleanupFinalizer); err != nil {
		return err
	}

	// delete all userprojectbindings related to user
	userProjectBindings, err := r.bindingsForUser(ctx, user)
	if err != nil {
		return err
	}
	for _, userBinding := range userProjectBindings {
		if err := r.masterClient.Delete(ctx, &userBinding); err != nil {
			if err := ctrlruntimeclient.IgnoreNotFound(err); err != nil {
				return err
			}
		}
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, user, masterUserProjectBindingCleanupFinalizer)
}

func (r *reconciler) bindingsForUser(ctx context.Context, user *kubermaticv1.User) ([]kubermaticv1.UserProjectBinding, error) {
	projectBindings := &kubermaticv1.UserProjectBindingList{}
	if err := r.masterClient.List(ctx, projectBindings); err != nil {
		return nil, err
	}

	var userProjectBindings []kubermaticv1.UserProjectBinding
	for _, projectBinding := range projectBindings.Items {
		if strings.EqualFold(user.Spec.Email, projectBinding.Spec.UserEmail) {
			userProjectBindings = append(userProjectBindings, projectBinding)
		}
	}

	return userProjectBindings, nil
}
