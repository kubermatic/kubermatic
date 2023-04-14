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
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	clusterhelper "k8c.io/kubermatic/v3/pkg/cluster"
	clusterupdatecontroller "k8c.io/kubermatic/v3/pkg/controller/seed-controller-manager/cluster-update-controller"
	kuberneteshelper "k8c.io/kubermatic/v3/pkg/kubernetes"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kkp-cluster-phase-controller"
)

type Reconciler struct {
	ctrlruntimeclient.Client

	recorder record.EventRecorder
	log      *zap.SugaredLogger
	versions kubermatic.Versions
}

// Add creates a new update controller.
func Add(mgr manager.Manager, numWorkers int, log *zap.SugaredLogger, versions kubermatic.Versions) error {
	reconciler := &Reconciler{
		Client: mgr.GetClient(),

		recorder: mgr.GetEventRecorderFor(ControllerName),
		log:      log,
		versions: versions,
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %w", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch: %w", err)
	}

	return nil
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
		log.Errorw("Failed to reconcile", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}

	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	// deletion timestamp overrides everything else
	if cluster.DeletionTimestamp != nil {
		return r.setClusterPhase(ctx, cluster, kubermaticv1.ClusterPhaseTerminating)
	}

	// if this cluster was never fully reconciled (yet), it is in Creating phase
	if !clusterhelper.IsClusterInitialized(cluster, r.versions) {
		return r.setClusterPhase(ctx, cluster, kubermaticv1.ClusterPhaseCreating)
	}

	// if there is a pending update condition, the cluster is in Updating phase
	cond, exists := cluster.Status.Conditions[kubermaticv1.ClusterConditionUpdateProgress]
	if exists && cond.Reason != clusterupdatecontroller.ClusterConditionUpToDate {
		return r.setClusterPhase(ctx, cluster, kubermaticv1.ClusterPhaseUpdating)
	}

	// in the absence of more smarter logic, every other status is just "Running"
	// (going to something like "Reconciling" whenever the control plane is unhealthy
	// might cause maaaaany status changes)
	return r.setClusterPhase(ctx, cluster, kubermaticv1.ClusterPhaseRunning)
}

func (r *Reconciler) setClusterPhase(ctx context.Context, cluster *kubermaticv1.Cluster, phase kubermaticv1.ClusterPhase) error {
	return kuberneteshelper.UpdateClusterStatus(ctx, r, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.Phase = phase
	})
}
