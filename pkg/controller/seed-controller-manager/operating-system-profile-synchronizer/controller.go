/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticpred "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"
	osmreconciling "k8c.io/operating-system-manager/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
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
	ControllerName = "kkp-operating-system-profile-synchronizer"

	// cleanupFinalizer indicates that the OperatingSystemProfile needs to be removed from all the user-cluster namespaces.
	cleanupFinalizer = "kubermatic.k8c.io/cleanup-kubermatic-operating-system-profiles"
)

type Reconciler struct {
	log                     *zap.SugaredLogger
	workerNameLabelSelector labels.Selector
	workerName              string
	recorder                record.EventRecorder
	namespace               string
	seedClient              ctrlruntimeclient.Client
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	namespace string,
	numWorkers int,
) error {
	workerSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return fmt.Errorf("failed to build worker-name selector: %w", err)
	}

	reconciler := &Reconciler{
		log:                     log.Named(ControllerName),
		workerNameLabelSelector: workerSelector,
		workerName:              workerName,
		recorder:                mgr.GetEventRecorderFor(ControllerName),
		namespace:               namespace,
		seedClient:              mgr.GetClient(),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	// Watch changes for OSPs.
	if err := c.Watch(
		&source.Kind{Type: &osmv1alpha1.OperatingSystemProfile{}},
		&handler.EnqueueRequestForObject{},
		kubermaticpred.ByNamespace(namespace),
	); err != nil {
		return fmt.Errorf("failed to create watch for operatingSystemProfiles: %w", err)
	}

	// Watch changes for OSPs and then enqueue all the clusters where OSM is enabled.
	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.Cluster{}},
		enqueueOperatingSystemProfiles(reconciler.seedClient, reconciler.log, namespace),
		workerlabel.Predicates(workerName),
		withEventFilter(),
	); err != nil {
		return fmt.Errorf("failed to create watch for clusters: %w", err)
	}
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("operatingsystemprofile", request.NamespacedName.String())
	log.Debug("Reconciling")

	osp := &osmv1alpha1.OperatingSystemProfile{}
	if err := r.seedClient.Get(ctx, request.NamespacedName, osp); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// OperatingSystemProfile is marked for deletion.
	if osp.DeletionTimestamp != nil {
		log.Debug("Deletion timestamp found for operatingSystemProfile")
		if kuberneteshelper.HasFinalizer(osp, cleanupFinalizer) {
			if err := r.handleDeletion(ctx, log, osp); err != nil {
				err = fmt.Errorf("failed to delete operatingSystemProfile: %w", err)

				log.Errorw("ReconcilingError", zap.Error(err))
				r.recorder.Event(osp, corev1.EventTypeWarning, "ReconcilingError", err.Error())

				return reconcile.Result{}, err
			}
			return reconcile.Result{}, nil
		}
		// Finalizer doesn't exist so clean up is already done.
		return reconcile.Result{}, nil
	}

	err := r.reconcile(ctx, log, osp)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(osp, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, osp *osmv1alpha1.OperatingSystemProfile) error {
	if err := kuberneteshelper.TryAddFinalizer(ctx, r.seedClient, osp, cleanupFinalizer); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}
	return r.syncAllUserClusterNamespaces(ctx, log, osp)
}

func (r *Reconciler) handleDeletion(ctx context.Context, log *zap.SugaredLogger, osp *osmv1alpha1.OperatingSystemProfile) error {
	err := r.syncAllUserClusterNamespaces(ctx, log, osp)
	if err != nil {
		return err
	}

	// Remove the finalizer
	return kuberneteshelper.TryRemoveFinalizer(ctx, r.seedClient, osp, cleanupFinalizer)
}

func (r *Reconciler) syncAllUserClusterNamespaces(ctx context.Context, log *zap.SugaredLogger, osp *osmv1alpha1.OperatingSystemProfile) error {
	clusters := &kubermaticv1.ClusterList{}
	if err := r.seedClient.List(ctx, clusters); err != nil {
		log.Error(err)
		utilruntime.HandleError(fmt.Errorf("failed to list clusters: %w", err))
	}

	var errors []error
	for _, cluster := range clusters.Items {
		// Ensure that this is a reconcilable cluster
		if cluster.Spec.IsOperatingSystemManagerEnabled() && cluster.DeletionTimestamp == nil && !cluster.Spec.Pause && cluster.Status.NamespaceName != "" {
			err := r.syncOperatingSystemProfile(ctx, log, osp, cluster.Status.NamespaceName)
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return kerrors.NewAggregate(errors)
}

func (r *Reconciler) syncOperatingSystemProfile(ctx context.Context, log *zap.SugaredLogger, osp *osmv1alpha1.OperatingSystemProfile, namespace string) error {
	// If OSP is marked for deletion then remove it from the user-cluster namespace.
	if osp.DeletionTimestamp != nil {
		toDelete := &osmv1alpha1.OperatingSystemProfile{}
		toDelete.Name = osp.Name
		toDelete.Namespace = namespace

		err := r.seedClient.Delete(ctx, toDelete)
		return ctrlruntimeclient.IgnoreNotFound(err)
	}

	creators := []reconciling.NamedOperatingSystemProfileReconcilerFactory{
		ospCreator(osp),
	}

	if err := osmreconciling.ReconcileOperatingSystemProfiles(ctx, creators, namespace, r.seedClient); err != nil {
		return fmt.Errorf("failed to reconcile OSP: %w", err)
	}

	return nil
}

func withEventFilter() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			cluster, ok := e.Object.(*kubermaticv1.Cluster)
			if !ok {
				return false
			}
			return cluster.Spec.IsOperatingSystemManagerEnabled() && cluster.DeletionTimestamp == nil && cluster.Status.NamespaceName != ""
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldCluster, ok := e.ObjectOld.(*kubermaticv1.Cluster)
			if !ok {
				return false
			}
			newCluster, ok := e.ObjectNew.(*kubermaticv1.Cluster)
			if !ok {
				return false
			}
			// We might need to install or delete custom OSPs from the user cluster namespace.
			if oldCluster.Spec.EnableOperatingSystemManager != newCluster.Spec.EnableOperatingSystemManager && newCluster.DeletionTimestamp == nil && newCluster.Status.NamespaceName != "" {
				return true
			}

			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}
}

func enqueueOperatingSystemProfiles(client ctrlruntimeclient.Client, log *zap.SugaredLogger, namespace string) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var requests []reconcile.Request

		ospList := &osmv1alpha1.OperatingSystemProfileList{}

		if err := client.List(context.Background(), ospList, &ctrlruntimeclient.ListOptions{Namespace: namespace}); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list operatingSystemProfiles: %w", err))
		}

		for _, osp := range ospList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Name:      osp.Name,
				Namespace: osp.Namespace,
			}})
		}
		return requests
	})
}

func ospCreator(osp *osmv1alpha1.OperatingSystemProfile) osmreconciling.NamedOperatingSystemProfileReconcilerFactory {
	return func() (string, osmreconciling.OperatingSystemProfileCreator) {
		return osp.Name, func(existing *osmv1alpha1.OperatingSystemProfile) (*osmv1alpha1.OperatingSystemProfile, error) {
			// We need to check if the existing OperatingSystemProfile can be updated.
			// OSP is immutable by nature and to make modifications a version bump is mandatory,
			// so we only update the OSP if the version is different.
			if existing.Spec.Version != osp.Spec.Version {
				existing.Spec = osp.Spec
			}

			return existing, nil
		}
	}
}
