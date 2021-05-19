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
	"time"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

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
	// This controller syncs the kubermatic constraints on the master cluster to the seed clusters.
	ControllerName = "master_constraint_syncing_controller"
)

type reconciler struct {
	log          *zap.SugaredLogger
	masterClient ctrlruntimeclient.Client
	namespace    string
	seedClients  map[string]ctrlruntimeclient.Client
	recorder     record.EventRecorder
}

func Add(ctx context.Context,
	masterMgr manager.Manager,
	namespace string,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger) error {

	log = log.Named(ControllerName)
	r := &reconciler{
		log:          log,
		masterClient: masterMgr.GetClient(),
		namespace:    namespace,
		seedClients:  map[string]ctrlruntimeclient.Client{},
		recorder:     masterMgr.GetEventRecorderFor(ControllerName),
	}

	c, err := controller.New(ControllerName, masterMgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()

		seedConstraintWatch := &source.Kind{Type: &kubermaticv1.Constraint{}}
		if err := seedConstraintWatch.InjectCache(seedManager.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache for seed %q into watch: %v", seedName, err)
		}

		if err := c.Watch(seedConstraintWatch, &handler.EnqueueRequestForObject{}, predicate.ByNamespace(namespace)); err != nil {
			return fmt.Errorf("failed to watch constraints in seed %q: %w", seedName, err)
		}
	}

	// Watch for changes to Constraints
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Constraint{}}, &handler.EnqueueRequestForObject{}, predicate.ByNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to watch constraints: %v", err)
	}

	return nil
}

func constraintCreatorGetter(constraint *kubermaticv1.Constraint) reconciling.NamedKubermaticV1ConstraintCreatorGetter {
	return func() (string, reconciling.KubermaticV1ConstraintCreator) {
		return constraint.Name, func(c *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
			c.Name = constraint.Name
			c.Spec = constraint.Spec
			c.Namespace = constraint.Namespace
			return c, nil
		}
	}
}

// Reconcile reconciles the kubermatic constraints in the master cluster and syncs them to all seeds
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request.Name)
	log.Debug("Processing")

	err := r.reconcile(ctx, log, request)
	if controllerutil.IsCacheNotStarted(err) {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
	}

	return reconcile.Result{}, err
}

func (r *reconciler) syncAllSeeds(ctx context.Context, log *zap.SugaredLogger, constraint *kubermaticv1.Constraint, action func(seedClient ctrlruntimeclient.Client, constraint *kubermaticv1.Constraint) error) error {
	for seedName, seedClient := range r.seedClients {

		log := log.With("seed", seedName)

		log.Debug("Reconciling constraint with seed")

		err := action(seedClient, constraint)
		if err != nil {
			return fmt.Errorf("failed syncing constraint %s for seed %s: %w", constraint.Name, seedName, err)
		}
		log.Debug("Reconciled constraint with seed")
	}
	return nil
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) error {

	constraint := &kubermaticv1.Constraint{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, constraint); err != nil {
		if controllerutil.IsCacheNotStarted(err) {
			return err
		}

		err := ctrlruntimeclient.IgnoreNotFound(err)

		return fmt.Errorf("failed to get constraint %s: %v", request.Name, err)
	}

	// handling deletion
	if !constraint.DeletionTimestamp.IsZero() {

		if err := r.handleDeletion(ctx, log, constraint); err != nil {
			return fmt.Errorf("handling deletion of constraint: %v", err)
		}
		return nil
	}

	if !kuberneteshelper.HasFinalizer(constraint, kubermaticapiv1.GatekeeperSeedConstraintCleanupFinalizer) {
		oldconstraint := constraint.DeepCopy()
		kuberneteshelper.AddFinalizer(constraint, kubermaticapiv1.GatekeeperSeedConstraintCleanupFinalizer)
		if err := r.masterClient.Patch(ctx, constraint, ctrlruntimeclient.MergeFrom(oldconstraint)); err != nil {
			return fmt.Errorf("failed to set constraint finalizer %s: %v", constraint.Name, err)
		}
	}

	constraintCreatorGetters := []reconciling.NamedKubermaticV1ConstraintCreatorGetter{
		constraintCreatorGetter(constraint),
	}

	return r.syncAllSeeds(ctx, log, constraint, func(seedClient ctrlruntimeclient.Client, constraint *kubermaticv1.Constraint) error {
		return reconciling.ReconcileKubermaticV1Constraints(ctx, constraintCreatorGetters, r.namespace, seedClient)
	})
}

func (r *reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, constraint *kubermaticv1.Constraint) error {

	// if finalizer not set to master Constraint then return
	if !kuberneteshelper.HasFinalizer(constraint, kubermaticapiv1.GatekeeperSeedConstraintCleanupFinalizer) {
		return nil
	}

	err := r.syncAllSeeds(ctx, log, constraint, func(seedClient ctrlruntimeclient.Client, constraint *kubermaticv1.Constraint) error {
		err := seedClient.Delete(ctx, &kubermaticv1.Constraint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constraint.Name,
				Namespace: constraint.Namespace,
			},
		})

		err = ctrlruntimeclient.IgnoreNotFound(err)

		return err
	})
	if err != nil {
		return err
	}

	oldconstraint := constraint.DeepCopy()
	kuberneteshelper.RemoveFinalizer(constraint, kubermaticapiv1.GatekeeperSeedConstraintCleanupFinalizer)
	if err := r.masterClient.Patch(ctx, constraint, ctrlruntimeclient.MergeFrom(oldconstraint)); err != nil {
		return fmt.Errorf("failed to remove constraint finalizer %s: %v", constraint.Name, err)
	}
	return nil
}
