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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// cleanupFinalizer indicates that the OperatingSystemProfile needs to be removed from all the user-cluster namespaces.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-kubermatic-operating-system-profiles"
	ospNamespace     = metav1.NamespaceSystem
)

// syncReconciler watches custom OSPs and syncs them into all existing user clusters
// whenever a custom OSP changes.
type syncReconciler struct {
	log                           *zap.SugaredLogger
	workerName                    string
	recorder                      events.EventRecorder
	namespace                     string
	seedClient                    ctrlruntimeclient.Client
	userClusterConnectionProvider UserClusterClientProvider
}

func addSyncReconciler(
	mgr manager.Manager,
	userClusterConnectionProvider UserClusterClientProvider,
	log *zap.SugaredLogger,
	workerName string,
	namespace string,
	numWorkers int,
) error {
	controllerName := controllerName("synchronizer")

	reconciler := &syncReconciler{
		log:                           log.Named(controllerName),
		workerName:                    workerName,
		recorder:                      mgr.GetEventRecorder(controllerName),
		namespace:                     namespace,
		userClusterConnectionProvider: userClusterConnectionProvider,
		seedClient:                    mgr.GetClient(),
	}

	customOSP := &unstructured.Unstructured{}
	customOSP.SetAPIVersion(operatingSystemManagerAPIVersion)
	customOSP.SetKind(customOperatingSystemProfileKind)

	_, err := builder.ControllerManagedBy(mgr).
		Named(controllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(customOSP, builder.WithPredicates(kubermaticpred.ByNamespace(namespace))).
		Build(reconciler)

	return err
}

func (r *syncReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("operatingsystemprofile", request.String())
	log.Debug("Reconciling")

	cosp := &unstructured.Unstructured{}
	cosp.SetAPIVersion(operatingSystemManagerAPIVersion)
	cosp.SetKind(customOperatingSystemProfileKind)

	if err := r.seedClient.Get(ctx, request.NamespacedName, cosp); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err := r.reconcile(ctx, log, cosp)
	if err != nil {
		r.recorder.Eventf(cosp, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *syncReconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cosp *unstructured.Unstructured) error {
	if cosp.GetDeletionTimestamp() != nil {
		return r.handleDeletion(ctx, log, cosp)
	}

	if err := kuberneteshelper.TryAddFinalizer(ctx, r.seedClient, cosp, cleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	return r.syncAllUserClusters(ctx, cosp)
}

func (r *syncReconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, cosp *unstructured.Unstructured) error {
	log.Debug("Deletion timestamp found for OperatingSystemProfile")

	if !kuberneteshelper.HasFinalizer(cosp, cleanupFinalizer) {
		return nil
	}

	err := r.syncAllUserClusters(ctx, cosp)
	if err != nil {
		return err
	}

	return kuberneteshelper.TryRemoveFinalizer(ctx, r.seedClient, cosp, cleanupFinalizer)
}

func (r *syncReconciler) syncAllUserClusters(ctx context.Context, cosp *unstructured.Unstructured) error {
	osp, err := customOSPToOSP(cosp)
	if err != nil {
		return err
	}

	clusters := &kubermaticv1.ClusterList{}
	if err := r.seedClient.List(ctx, clusters); err != nil {
		return fmt.Errorf("failed to list clusters: %w", err)
	}

	var errors []error
	for _, cluster := range clusters.Items {
		// Ensure that this is a reconcilable cluster
		if r.syncableCluster(&cluster) {
			err := r.syncOperatingSystemProfile(ctx, osp, &cluster)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return kerrors.NewAggregate(errors)
}

func (r *syncReconciler) syncableCluster(cluster *kubermaticv1.Cluster) bool {
	return true &&
		!cluster.Spec.Pause &&
		cluster.Labels[kubermaticv1.WorkerNameLabelKey] == r.workerName &&
		cluster.DeletionTimestamp == nil &&
		cluster.Status.ExtendedHealth.ControlPlaneHealthy()
}

func (r *syncReconciler) syncOperatingSystemProfile(ctx context.Context, osp *osmv1alpha1.OperatingSystemProfile, cluster *kubermaticv1.Cluster) error {
	userClusterClient, err := r.userClusterConnectionProvider.GetClient(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get user cluster client: %w", err)
	}

	// If OSP is marked for deletion then remove it from the user cluster.
	if osp.DeletionTimestamp != nil {
		toDelete := &osmv1alpha1.OperatingSystemProfile{}
		toDelete.Name = osp.Name
		toDelete.Namespace = osp.Namespace

		return ctrlruntimeclient.IgnoreNotFound(userClusterClient.Delete(ctx, toDelete))
	}

	creators := []reconciling.NamedOperatingSystemProfileReconcilerFactory{
		ospReconciler(osp),
	}

	if err := reconciling.ReconcileOperatingSystemProfiles(ctx, creators, ospNamespace, userClusterClient); err != nil {
		return fmt.Errorf("failed to reconcile OSP: %w", err)
	}

	return nil
}
