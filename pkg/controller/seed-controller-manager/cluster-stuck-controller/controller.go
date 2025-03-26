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

package clusterstuckcontroller

import (
	"context"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-cluster-stuck-controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	workerName string
	recorder   record.EventRecorder
	log        *zap.SugaredLogger
}

// Add creates a new cluster-stuck controller.
func Add(mgr manager.Manager, numWorkers int, workerName string, log *zap.SugaredLogger) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		workerName: workerName,
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		log:        log,
	}

	_, err := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		}).
		For(&kubermaticv1.Cluster{}).
		Build(reconciler)

	return err
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("cluster", request.Name)
	log.Debug("Reconciling")

	// If the seed-ctrl-mgr is itself controlled by a worker, it should
	// not influence non-workered clusters.
	if r.workerName != "" {
		return reconcile.Result{}, nil
	}

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		return reconcile.Result{}, ctrlruntimeclient.IgnoreNotFound(err)
	}

	// not deleted
	if cluster.DeletionTimestamp == nil {
		return reconcile.Result{}, nil
	}

	// paused clusters will not be removed
	if cluster.Spec.Pause {
		reason := cluster.Spec.PauseReason
		if reason == "" {
			reason = "no reason given"
		}

		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ClusterPaused", "Cluster cannot be cleaned up because it is paused: %s", reason)
	}

	// no worker name
	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != "" {
		// cluster seems stuck (we wait for a bit because we cannot easily
		// tell if a seed-ctrl-mgr with the given worker-name is actually
		// up and running right now)
		if time.Since(cluster.DeletionTimestamp.Time) > 5*time.Minute {
			r.recorder.Eventf(cluster, corev1.EventTypeWarning, "WorkerName", "A %s label is set, preventing the regular seed-controller-manager from cleaning up.", kubermaticv1.WorkerNameLabelKey)
		}
	}

	// renew the event to keep it visible
	return reconcile.Result{RequeueAfter: 5 * time.Minute}, nil
}
