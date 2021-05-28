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

package seedconstraintsynchronizer

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	controllerutil "k8c.io/kubermatic/v2/pkg/controller/util"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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
	// This controller syncs the kubermatic constraints to constraint on the user cluster.
	ControllerName = "constraint_syncing_controller"
	finalizer      = kubermaticapiv1.KubermaticUserClusterNsDefaultConstraintCleanupFinalizer
	Key            = "default"
	Value          = "true"
	AddAction      = "add"
	RemoveAction   = "remove"
)

type reconciler struct {
	log                     *zap.SugaredLogger
	workerNameLabelSelector labels.Selector
	recorder                record.EventRecorder
	namespace               string
	seedClient              ctrlruntimeclient.Client
}

func opaPredicate() predicate.Funcs {
	return kubermaticpred.Factory(func(o ctrlruntimeclient.Object) bool {
		cluster, ok := o.(*kubermaticv1.Cluster)
		if !ok {
			return false
		}
		return cluster.Spec.OPAIntegration != nil && cluster.Spec.OPAIntegration.Enabled
	})
}

func withEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldConstraint, ok := e.ObjectOld.(*kubermaticv1.Constraint)
			if !ok {
				return false
			}
			newConstraint, ok := e.ObjectNew.(*kubermaticv1.Constraint)
			if !ok {
				return false
			}
			return !reflect.DeepEqual(oldConstraint.Spec, newConstraint.Spec)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func Add(ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	namespace string,
	numWorkers int) error {

	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %v", err)
	}

	reconciler := &reconciler{
		log:                     log.Named(ControllerName),
		workerNameLabelSelector: workerSelector,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		namespace:               namespace,
		seedClient:              mgr.GetClient(),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Cluster{}},
		enqueueConstraints(reconciler.seedClient, reconciler.log, namespace),
		workerlabel.Predicates(workerName),
		opaPredicate(),
	); err != nil {
		return fmt.Errorf("failed to create watch for clusters: %w", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Constraint{}},
		&handler.EnqueueRequestForObject{},
		kubermaticpred.ByNamespace(namespace),
	); err != nil {
		return fmt.Errorf("failed to create watch for seed constraints: %v", err)
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Constraint{}},
		handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
			return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: a.GetName(), Namespace: namespace}}}
		}),
		kubermaticpred.ByLabel(Key, Value),
		withEventFilter(),
	); err != nil {
		return fmt.Errorf("failed to create watch for user cluster namespace constraints: %v", err)
	}

	return nil
}

// Reconcile reconciles the kubermatic constraints in the seed cluster and syncs them to all user clusters namespace
// which have opa integration enabled
func (r *reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Reconciling")

	constraint := &kubermaticv1.Constraint{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, constraint); err != nil {
		if controllerutil.IsCacheNotStarted(err) {
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get constraint %s: %v", constraint.Name, ctrlruntimeclient.IgnoreNotFound(err))

	}

	err := r.reconcile(ctx, constraint, log)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
		r.recorder.Eventf(constraint, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func addLabel(constraint *kubermaticv1.Constraint) *kubermaticv1.Constraint {
	if constraint.Labels != nil {
		constraint.Labels[Key] = Value
	} else {
		constraint.Labels = map[string]string{Key: Value}
	}
	return constraint
}

func constraintCreatorGetter(constraint *kubermaticv1.Constraint) reconciling.NamedKubermaticV1ConstraintCreatorGetter {
	return func() (string, reconciling.KubermaticV1ConstraintCreator) {
		return constraint.Name, func(c *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
			c.Name = constraint.Name
			c.Spec = constraint.Spec
			c = addLabel(c)
			return c, nil
		}
	}
}

func (r *reconciler) patchFinalizer(ctx context.Context, constraint *kubermaticv1.Constraint, action string) error {
	oldconstraint := constraint.DeepCopy()

	if action == AddAction {
		kuberneteshelper.AddFinalizer(constraint, finalizer)
	} else if action == RemoveAction {
		kuberneteshelper.RemoveFinalizer(constraint, finalizer)
	}

	if err := r.seedClient.Patch(ctx, constraint, ctrlruntimeclient.MergeFrom(oldconstraint)); err != nil {
		return fmt.Errorf("failed to %s constraint finalizer %s: %v", action, constraint.Name, err)
	}

	return nil
}

func (r *reconciler) reconcile(ctx context.Context, constraint *kubermaticv1.Constraint, log *zap.SugaredLogger) error {

	// constraint deletion
	if !constraint.DeletionTimestamp.IsZero() {
		if !kuberneteshelper.HasFinalizer(constraint, finalizer) {
			return nil
		}

		if err := r.cleanupConstraint(ctx, log, constraint); err != nil {
			return err
		}
		return nil
	}

	// constraint initialization
	if !kuberneteshelper.HasFinalizer(constraint, finalizer) {
		if err := r.patchFinalizer(ctx, constraint, AddAction); err != nil {
			return err
		}
	}

	// constraint creation
	constraintCreatorGetters := []reconciling.NamedKubermaticV1ConstraintCreatorGetter{
		constraintCreatorGetter(constraint),
	}

	return r.syncAllClustersNS(ctx, log, constraint, func(seedClient ctrlruntimeclient.Client, constraint *kubermaticv1.Constraint, namespace string) error {
		return reconciling.ReconcileKubermaticV1Constraints(ctx, constraintCreatorGetters, namespace, seedClient)
	})
}

func (r *reconciler) cleanupConstraint(ctx context.Context, log *zap.SugaredLogger, constraint *kubermaticv1.Constraint) error {

	if err := r.syncAllClustersNS(ctx, log, constraint, func(seedClient ctrlruntimeclient.Client, constraint *kubermaticv1.Constraint, namespace string) error {

		log := log.With("constraint", constraint)
		log.Debugw("cleanup processing:", namespace)

		err := seedClient.Delete(ctx, &kubermaticv1.Constraint{
			ObjectMeta: metav1.ObjectMeta{
				Name:      constraint.Name,
				Namespace: namespace,
			},
		})
		return ctrlruntimeclient.IgnoreNotFound(err)

	}); err != nil {
		return err
	}

	if err := r.patchFinalizer(ctx, constraint, RemoveAction); err != nil {
		return err
	}
	return nil
}

func isOPAEnabled(userCluster *kubermaticv1.Cluster) bool {
	return userCluster.Spec.OPAIntegration != nil && userCluster.Spec.OPAIntegration.Enabled
}

func (r *reconciler) syncAllClustersNS(
	ctx context.Context,
	log *zap.SugaredLogger,
	constraint *kubermaticv1.Constraint,
	actionFunc func(seedClient ctrlruntimeclient.Client, constraint *kubermaticv1.Constraint, namespace string) error) error {

	clusterList := &kubermaticv1.ClusterList{}
	if err := r.seedClient.List(ctx, clusterList, &ctrlruntimeclient.ListOptions{LabelSelector: r.workerNameLabelSelector}); err != nil {
		return fmt.Errorf("failed listing clusters: %w", err)
	}

	for _, userCluster := range clusterList.Items {

		clusterName := userCluster.Spec.HumanReadableName

		// instance Validation
		if userCluster.Spec.Pause {
			log.Debugw("Cluster paused, skipping", "cluster", clusterName)
			continue
		}

		if isOPAEnabled(&userCluster) {
			if err := actionFunc(r.seedClient, constraint, userCluster.Status.NamespaceName); err != nil {
				return fmt.Errorf("failed syncing constraint for cluster %s namespace: %w", clusterName, err)
			}

			log.Debugw("Reconciled constraint with cluster", "cluster", clusterName)
		} else {
			log.Debugw("Cluster does not integrate with OPA, skipping", "cluster", clusterName)
		}
	}

	return nil
}

func enqueueConstraints(client ctrlruntimeclient.Client, log *zap.SugaredLogger, namespace string) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		constraintList := &kubermaticv1.ConstraintList{}
		if err := client.List(context.Background(), constraintList, &ctrlruntimeclient.ListOptions{Namespace: namespace}); err != nil {
			log.Error(err)
			utilruntime.HandleError(fmt.Errorf("failed to list constraints: %v", err))
		}

		for _, constraint := range constraintList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      constraint.Name,
				Namespace: constraint.Namespace,
			}})
		}
		return requests
	})
}
