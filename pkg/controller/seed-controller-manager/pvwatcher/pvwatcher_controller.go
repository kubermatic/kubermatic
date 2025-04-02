/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package pvwatcher

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	predicateutils "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-persistent-volume-watcher"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	log        *zap.SugaredLogger
	workerName string
	recorder   record.EventRecorder
}

// add the controller.
func Add(
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	workerName string,
) error {
	log = log.Named(ControllerName)
	reconciler := &Reconciler{
		Client:     mgr.GetClient(),
		log:        log,
		workerName: workerName,
		recorder:   mgr.GetEventRecorderFor(ControllerName),
	}

	// reconcile PVCs in ClaimLost phase only
	lostClaimPredicates := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			newObj := e.ObjectNew.(*corev1.PersistentVolumeClaim)
			return newObj.Status.Phase == corev1.ClaimLost
		},
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&corev1.PersistentVolumeClaim{}, builder.WithPredicates(lostClaimPredicates, predicateutils.ByLabel(resources.AppLabelKey, resources.EtcdStatefulSetName))).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("pvc", request)
	log.Debug("Reconciling")

	cluster, err := kubernetes.ClusterFromNamespace(ctx, r, request.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}
	if cluster == nil {
		return reconcile.Result{}, nil
	}
	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName || cluster.Spec.Pause {
		return reconcile.Result{}, nil
	}
	if !cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] {
		return reconcile.Result{}, nil
	}
	result, err := r.reconcile(ctx, log, request)
	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) (reconcile.Result, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, pvc); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if pvc.Status.Phase != corev1.ClaimLost {
		return reconcile.Result{}, nil
	}
	// find the pvc pod, delete it
	podName := strings.ReplaceAll(pvc.Name, "data-", "")
	pvcPod := &corev1.Pod{}
	if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: pvc.Namespace}, pvcPod); err != nil {
		return reconcile.Result{}, err
	}
	if err := r.Delete(ctx, pvcPod); err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	// When the pod is deleted and recreated it, it will be stuck in PodPending phase
	if err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
		if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: pvc.Namespace}, pvcPod); err != nil {
			return false, err
		}
		return pvcPod.Status.Phase == corev1.PodPending, nil
	}); err != nil {
		return reconcile.Result{}, err
	}

	// delete the pvc, make sure it's deleted
	if err := r.Delete(ctx, pvc); err != nil && !apierrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	if err := wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Second, false, func(ctx context.Context) (bool, error) {
		err := r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, pvc)
		if err != nil && apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, err
	}); err != nil {
		return reconcile.Result{}, err
	}
	// Workaround to force the sts to recreate the PVC/PV, we need to "reboot" the StatefulSet.
	return reconcile.Result{},
		r.DeleteAllOf(ctx, &corev1.Pod{}, ctrlruntimeclient.InNamespace(pvc.Namespace),
			ctrlruntimeclient.MatchingLabels{resources.AppLabelKey: resources.EtcdStatefulSetName})
}
