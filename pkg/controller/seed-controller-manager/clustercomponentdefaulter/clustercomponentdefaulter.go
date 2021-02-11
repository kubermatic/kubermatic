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

package clustercomponentdefaulter

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "clustercomponent_defaulter"

type Reconciler struct {
	log        *zap.SugaredLogger
	client     ctrlruntimeclient.Client
	recorder   record.EventRecorder
	defaults   kubermaticv1.ComponentSettings
	workerName string
	versions   kubermatic.Versions
}

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	defaults kubermaticv1.ComponentSettings,
	workerName string,
	versions kubermatic.Versions) error {

	reconciler := &Reconciler{
		log:        log.Named(ControllerName),
		client:     mgr.GetClient(),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		defaults:   defaults,
		workerName: workerName,
		versions:   versions,
	}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.client.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}
	log = r.log.With("cluster", cluster.Name)

	// Add a wrapping here so we can emit an event on error
	_, err := kubermaticv1helper.ClusterReconcileWrapper(
		context.Background(),
		r.client,
		r.workerName,
		cluster,
		r.versions,
		kubermaticv1.ClusterConditionComponentDefaulterReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return nil, r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		r.log.With("error", err).With("cluster", request.NamespacedName.Name).Error("failed to reconcile cluster")
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	log.Debug("Syncing cluster")

	targetComponentsOverride := cluster.Spec.ComponentsOverride.DeepCopy()
	if targetComponentsOverride.Apiserver.Replicas == nil {
		targetComponentsOverride.Apiserver.Replicas = r.defaults.Apiserver.Replicas
	}
	if targetComponentsOverride.Apiserver.Resources == nil {
		targetComponentsOverride.Apiserver.Resources = r.defaults.Apiserver.Resources
	}
	if targetComponentsOverride.Apiserver.EndpointReconcilingDisabled == nil {
		targetComponentsOverride.Apiserver.EndpointReconcilingDisabled = r.defaults.Apiserver.EndpointReconcilingDisabled
	}
	if targetComponentsOverride.ControllerManager.Replicas == nil {
		targetComponentsOverride.ControllerManager.Replicas = r.defaults.ControllerManager.Replicas
	}
	if targetComponentsOverride.ControllerManager.Resources == nil {
		targetComponentsOverride.ControllerManager.Resources = r.defaults.ControllerManager.Resources
	}
	if targetComponentsOverride.Scheduler.Replicas == nil {
		targetComponentsOverride.Scheduler.Replicas = r.defaults.Scheduler.Replicas
	}
	if targetComponentsOverride.Scheduler.Resources == nil {
		targetComponentsOverride.Scheduler.Resources = r.defaults.Scheduler.Resources
	}
	if targetComponentsOverride.Etcd.Resources == nil {
		targetComponentsOverride.Etcd.Resources = r.defaults.Etcd.Resources
	}
	if targetComponentsOverride.Prometheus.Resources == nil {
		targetComponentsOverride.Prometheus.Resources = r.defaults.Prometheus.Resources
	}

	if apiequality.Semantic.DeepEqual(&cluster.Spec.ComponentsOverride, targetComponentsOverride) {
		return nil
	}

	oldCluster := cluster.DeepCopy()
	targetComponentsOverride.DeepCopyInto(&cluster.Spec.ComponentsOverride)
	if err := r.client.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to update componentsOverride: %v", err)
	}
	log.Info("Successfully defaulted componentsOverride")
	return nil
}
