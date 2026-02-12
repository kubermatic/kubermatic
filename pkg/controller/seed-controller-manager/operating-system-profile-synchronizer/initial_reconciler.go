/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package operatingsystemprofilesynchronizer

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// clusterInitReconciler is used to watch for new clusters and to install the
// initial set of OSPs into them. This is more involved than simply filtering
// for CREATE events only, since we need to wait for the control plane to
// actually become ready. So this reconciler will just listen for CREATE events
// *and* then requeue a cluster if it's not actually ready yet. Once the initial
// setup is done, the cluster is no longer requeued.
// We use a dedicated reconciler instead of a condition on the cluster object
// because this is simply a performance optimization to avoid having to install
// OSPs into all clustes whenever a cluster changes.
type clusterInitReconciler struct {
	seedClient                    ctrlruntimeclient.Client
	log                           *zap.SugaredLogger
	namespace                     string
	recorder                      events.EventRecorder
	userClusterConnectionProvider UserClusterClientProvider
}

func addClusterInitReconciler(
	seedMgr manager.Manager,
	userClusterConnectionProvider UserClusterClientProvider,
	log *zap.SugaredLogger,
	workerName string,
	namespace string,
	numWorkers int,
) error {
	controllerName := controllerName("initial-controller")

	reconciler := &clusterInitReconciler{
		seedClient:                    seedMgr.GetClient(),
		recorder:                      seedMgr.GetEventRecorder(controllerName),
		log:                           log.Named(controllerName),
		userClusterConnectionProvider: userClusterConnectionProvider,
		namespace:                     namespace,
	}

	_, err := builder.ControllerManagedBy(seedMgr).
		Named(controllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(
			&kubermaticv1.Cluster{},
			builder.WithPredicates(workerlabel.Predicate(workerName), clusterFilter()),
		).
		Build(reconciler)

	return err
}

func clusterFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func (r *clusterInitReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	cluster := &kubermaticv1.Cluster{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.DeletionTimestamp != nil || cluster.Spec.Pause {
		return reconcile.Result{}, nil
	}

	// do not even try the initial setup before the control plane is healthy
	if !cluster.Status.ExtendedHealth.ControlPlaneHealthy() {
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	err := r.reconcile(ctx, cluster)
	if err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	// If the reconciling failed, we return an error here, which will make controller-runtime
	// requeue the request.
	return reconcile.Result{}, err
}

func (r *clusterInitReconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	ospList := &unstructured.UnstructuredList{}
	ospList.SetAPIVersion(operatingSystemManagerAPIVersion)
	ospList.SetKind(fmt.Sprintf("%sList", customOperatingSystemProfileKind))

	if err := r.seedClient.List(ctx, ospList, &ctrlruntimeclient.ListOptions{Namespace: r.namespace}); err != nil {
		return fmt.Errorf("failed to list CustomOperatingSystemProfiles: %w", err)
	}

	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	factories := []reconciling.NamedOperatingSystemProfileReconcilerFactory{}
	for _, unstructuredOSP := range ospList.Items {
		// If OSP is marked for deletion then remove it from the user cluster.
		if unstructuredOSP.GetDeletionTimestamp() != nil {
			toDelete := &osmv1alpha1.OperatingSystemProfile{}
			toDelete.Name = unstructuredOSP.GetName()
			toDelete.Namespace = unstructuredOSP.GetNamespace()

			if err := userClusterClient.Delete(ctx, toDelete); ctrlruntimeclient.IgnoreNotFound(err) != nil {
				return fmt.Errorf("failed to delete OSP: %w", err)
			}
		} else {
			osp, err := customOSPToOSP(&unstructuredOSP)
			if err != nil {
				return err
			}

			factories = append(factories, ospReconciler(osp))
		}
	}

	if err := reconciling.ReconcileOperatingSystemProfiles(ctx, factories, ospNamespace, userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile OSPs: %w", err)
	}

	return nil
}
