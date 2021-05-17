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
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// This controller syncs the kubermatic constraint on the master cluster to the seed clusters.
	controllerName = "master_constraint_syncing_controller"
)

type reconciler struct {
	log              *zap.SugaredLogger
	recorder         record.EventRecorder
	masterClient     ctrlruntimeclient.Client
	namespace        string
	seedClientGetter provider.SeedClientGetter
}

func Add(ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	namespace string,
	seedKubeconfigGetter provider.SeedKubeconfigGetter) error {

	r := &reconciler{
		log:              log.Named(controllerName),
		masterClient:     mgr.GetClient(),
		namespace:        namespace,
		seedClientGetter: provider.SeedClientGetterFactory(seedKubeconfigGetter),
		recorder:         mgr.GetEventRecorderFor(controllerName),
	}

	c, err := controller.New(controllerName, mgr, controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	// Watch for changes to Constraints
	if err = c.Watch(
		&source.Kind{Type: &kubermaticv1.Constraint{}},
		&handler.EnqueueRequestForObject{},
		predicate.ByNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to create watch for the Constraints %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Seed{}},
		enqueueAllConstraint(r.masterClient, r.log),
		predicate.ByNamespace(namespace),
	); err != nil {
		return fmt.Errorf("failed to create seed watcher: %v", err)
	}

	return nil
}

// Reconcile reconciles the kubermatic constraints in the master cluster and syncs them to all seeds
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("resource", request)
	log.Debug("Reconciling")

	constraint := &kubermaticv1.Constraint{}
	if err := r.masterClient.Get(ctx, request.NamespacedName, constraint); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("constraint not found, returning")
			return reconcile.Result{}, nil
		}
		if controllerutil.IsCacheNotStarted(err) {
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get constraint %s: %v", constraint.Name, err)
	}

	err := r.reconcile(ctx, log, constraint)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(constraint, corev1.EventTypeWarning, "ConstraintReconcileFailed", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, constraint *kubermaticv1.Constraint) error {
	if constraint.DeletionTimestamp != nil {
		if !kuberneteshelper.HasFinalizer(constraint, kubermaticapiv1.GatekeeperSeedConstraintCleanupFinalizer) {
			return nil
		}

		err := r.syncAllSeeds(ctx, log, constraint, func(seedClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.Constraint) error {
			err := seedClusterClient.Delete(ctx, &kubermaticv1.Constraint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      constraint.Name,
					Namespace: constraint.Namespace,
				},
			})

			if kerrors.IsNotFound(err) {
				log.Debug("constraint not found, returning")
				return nil
			}

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

	if !kuberneteshelper.HasFinalizer(constraint, kubermaticapiv1.GatekeeperSeedConstraintCleanupFinalizer) {
		oldconstraint := constraint.DeepCopy()
		kuberneteshelper.AddFinalizer(constraint, kubermaticapiv1.GatekeeperSeedConstraintCleanupFinalizer)
		if err := r.masterClient.Patch(ctx, constraint, ctrlruntimeclient.MergeFrom(oldconstraint)); err != nil {
			return fmt.Errorf("failed to set constraint  finalizer %s: %v", constraint.Name, err)
		}
	}

	constraintCreatorGetters := []reconciling.NamedKubermaticV1ConstraintCreatorGetter{
		constraintCreatorGetter(constraint),
	}

	return r.syncAllSeeds(ctx, log, constraint, func(seedClusterClient ctrlruntimeclient.Client, ct *kubermaticv1.Constraint) error {
		return reconciling.ReconcileKubermaticV1Constraints(ctx, constraintCreatorGetters, r.namespace, seedClusterClient)
	})
}

func (r *reconciler) syncAllSeeds(
	ctx context.Context,
	log *zap.SugaredLogger,
	constraint *kubermaticv1.Constraint,
	action func(seedClusterClient ctrlruntimeclient.Client, c *kubermaticv1.Constraint) error) error {

	seedList := &kubermaticv1.SeedList{}
	if err := r.masterClient.List(ctx, seedList, &ctrlruntimeclient.ListOptions{Namespace: r.namespace}); err != nil {
		return fmt.Errorf("failed listing seeds: %w", err)
	}

	for _, seed := range seedList.Items {
		seedClient, err := r.seedClientGetter(&seed)
		if err != nil {
			return fmt.Errorf("failed getting seed client for seed %s: %w", seed.Name, err)
		}

		err = action(seedClient, constraint)
		if err != nil {
			return fmt.Errorf("failed syncing constraint for seed %s: %w", seed.Name, err)
		}
		log.Debugw("Reconciled constraint with seed", "seed", seed.Name)
	}

	return nil
}

func constraintCreatorGetter(kubeCT *kubermaticv1.Constraint) reconciling.NamedKubermaticV1ConstraintCreatorGetter {
	return func() (string, reconciling.KubermaticV1ConstraintCreator) {
		return kubeCT.Name, func(c *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
			c.Name = kubeCT.Name
			c.Spec = kubeCT.Spec
			c.Namespace = kubeCT.Namespace
			return c, nil
		}
	}
}

func enqueueAllConstraint(client ctrlruntimeclient.Client, log *zap.SugaredLogger) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		cList := &kubermaticv1.ConstraintList{}
		if err := client.List(context.Background(), cList); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list constraint: %v", err))
		}
		for _, c := range cList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      c.Name,
				Namespace: c.Namespace,
			}})
		}
		return requests
	})
}
