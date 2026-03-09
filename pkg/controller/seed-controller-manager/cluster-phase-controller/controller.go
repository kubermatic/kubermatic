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

package clusterphasecontroller

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	updatecontroller "k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/update-controller"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	ControllerName = "kkp-cluster-phase-controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	recorder events.EventRecorder
	log      *zap.SugaredLogger
	versions kubermatic.Versions
}

// Add creates a new update controller.
func Add(mgr manager.Manager, numWorkers int, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		recorder: mgr.GetEventRecorder(ControllerName),
		log:      log,
		versions: versions,
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

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	err := r.reconcile(ctx, log, cluster)
	if err != nil {
		r.recorder.Eventf(cluster, nil, corev1.EventTypeWarning, "ReconcilingError", "Reconciling", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	// deletion timestamp overrides everything else
	if cluster.DeletionTimestamp != nil {
		return r.setClusterPhase(ctx, cluster, kubermaticv1.ClusterTerminating)
	}

	// if this cluster was never fully reconciled (yet), it is in Creating phase
	if !util.IsClusterInitialized(cluster, r.versions) {
		return r.setClusterPhase(ctx, cluster, kubermaticv1.ClusterCreating)
	}

	// if there is a pending update condition, the cluster is in Updating phase
	cond, exists := cluster.Status.Conditions[kubermaticv1.ClusterConditionUpdateProgress]
	if exists && cond.Reason != updatecontroller.ClusterConditionUpToDate {
		return r.setClusterPhase(ctx, cluster, kubermaticv1.ClusterUpdating)
	}

	// in the absence of more smarter logic, every other status is just "Running"
	// (going to something like "Reconciling" whenever the control plane is unhealthy
	// might cause maaaaany status changes)
	return r.setClusterPhase(ctx, cluster, kubermaticv1.ClusterRunning)
}

func (r *Reconciler) setClusterPhase(ctx context.Context, cluster *kubermaticv1.Cluster, phase kubermaticv1.ClusterPhase) error {
	return util.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.Phase = phase
	})
}
