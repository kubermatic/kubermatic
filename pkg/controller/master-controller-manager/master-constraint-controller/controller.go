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

package masterconstraintsynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller syncs the kubermatic constraints on the master cluster to the seed clusters.
	ControllerName = "kkp-master-constraint-synchronizer"

	// cleanupFinalizer indicates that synced gatekeeper Constraint on seed clusters need cleanup.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-gatekeeper-seed-constraint"
)

type reconciler struct {
	log          *zap.SugaredLogger
	masterClient ctrlruntimeclient.Client
	namespace    string
	seedClients  kuberneteshelper.SeedClientMap
	recorder     record.EventRecorder
}

func Add(
	masterMgr manager.Manager,
	namespace string,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
) error {
	log = log.Named(ControllerName)
	r := &reconciler{
		log:          log,
		masterClient: masterMgr.GetClient(),
		namespace:    namespace,
		seedClients:  kuberneteshelper.SeedClientMap{},
		recorder:     masterMgr.GetEventRecorderFor(ControllerName),
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()
	}

	_, err := builder.ControllerManagedBy(masterMgr).
		Named(ControllerName).
		For(&kubermaticv1.Constraint{}, builder.WithPredicates(predicate.ByNamespace(namespace))).
		Build(r)

	return err
}

func constraintReconcilerFactory(constraint *kubermaticv1.Constraint) reconciling.NamedConstraintReconcilerFactory {
	return func() (string, reconciling.ConstraintReconciler) {
		return constraint.Name, func(c *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
			c.Name = constraint.Name
			c.Spec = constraint.Spec
			c.Namespace = constraint.Namespace
			return c, nil
		}
	}
}

// Reconcile reconciles the kubermatic constraints in the master cluster and syncs them to all seeds.
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("constraint", request.Name)
	log.Debug("Processing")

	constraint := &kubermaticv1.Constraint{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, constraint); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	err := r.reconcile(ctx, log, constraint)
	if err != nil {
		r.recorder.Event(constraint, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, constraint *kubermaticv1.Constraint) error {
	// handling deletion
	if !constraint.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, log, constraint); err != nil {
			return fmt.Errorf("handling deletion of constraint: %w", err)
		}
		return nil
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.masterClient, constraint, cleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	constraintReconcilerFactories := []reconciling.NamedConstraintReconcilerFactory{
		constraintReconcilerFactory(constraint),
	}

	return r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		log.Debug("Reconciling constraint with seed")

		seedConst := &kubermaticv1.Constraint{}
		if err := seedClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(constraint), seedConst); err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to fetch Constraint on seed cluster: %w", err)
		}

		// see project-synchronizer's syncAllSeeds comment
		if seedConst.UID != "" && seedConst.UID == constraint.UID {
			return nil
		}

		return reconciling.ReconcileConstraints(ctx, constraintReconcilerFactories, r.namespace, seedClient)
	})
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, constraint *kubermaticv1.Constraint) error {
	// if finalizer not set to master Constraint then return
	if !kuberneteshelper.HasFinalizer(constraint, cleanupFinalizer) {
		return nil
	}

	err := r.seedClients.Each(ctx, log, func(_ string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
		log.Debug("Deleting Constraint on Seed")

		err := seedClient.Delete(ctx, &kubermaticv1.Constraint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constraint.Name,
				Namespace: constraint.Namespace,
			},
		})

		return ctrlruntimeclient.IgnoreNotFound(err)
	})
	if err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.masterClient, constraint, cleanupFinalizer)
}
