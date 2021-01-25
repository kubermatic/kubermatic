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
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	predicateutils "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
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
	ControllerName = "kubermatic_volume_watcher_controller"
)

type Reconciler struct {
	log        *zap.SugaredLogger
	workerName string
	ctrlruntimeclient.Client
	recorder record.EventRecorder
}

// add the controller
func Add(

	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	workerName string,
) error {
	log = log.Named(ControllerName)
	reconciler := &Reconciler{
		log:        log,
		workerName: workerName,
		Client:     mgr.GetClient(),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}
	// reconcile PVCs in ClaimLost phase only
	LostClaimPredicates := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			new := e.ObjectNew.(*corev1.PersistentVolumeClaim)
			return new.Status.Phase == corev1.ClaimLost
		},
	}

	if err := c.Watch(&source.Kind{Type: &corev1.PersistentVolumeClaim{}}, &handler.EnqueueRequestForObject{},
		LostClaimPredicates,
		predicateutils.ByLabel(resources.AppLabelKey, resources.EtcdStatefulSetName)); err != nil {
		return fmt.Errorf("failed to create watch for PersistentVolumeClaims: %v", err)
	}
	return nil
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	clusterName := strings.ReplaceAll(request.Namespace, "cluster-", "")
	if err := r.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Skipping because the cluster is already gone")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName || cluster.Spec.Pause {
		return reconcile.Result{}, nil
	}
	if !cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] {
		return reconcile.Result{}, nil
	}
	result, err := r.reconcile(ctx, log, request)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, request reconcile.Request) (reconcile.Result, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Name: request.Name, Namespace: request.Namespace}, pvc); err != nil {
		if kerrors.IsNotFound(err) {
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
	if err := r.Delete(ctx, pvcPod); err != nil && !kerrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	// When the pod is deleted and recreated it, it will be stuck in PodPending phase
	if err := wait.Poll(5*time.Second, 30*time.Second, func() (bool, error) {
		if err := r.Get(ctx, types.NamespacedName{Name: podName, Namespace: pvc.Namespace}, pvcPod); err != nil {
			return false, err
		}
		return pvcPod.Status.Phase == corev1.PodPending, nil
	}); err != nil {
		return reconcile.Result{}, err
	}

	// delete the pvc, make sure it's deleted
	if err := r.Delete(ctx, pvc); err != nil && !kerrors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	if err := wait.Poll(5*time.Second, 30*time.Second, func() (bool, error) {
		err := r.Get(ctx, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, pvc)
		if err != nil && kerrors.IsNotFound(err) {
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
