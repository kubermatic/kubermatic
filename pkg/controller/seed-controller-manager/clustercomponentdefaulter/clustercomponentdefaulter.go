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
	"k8c.io/kubermatic/v2/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	utilpointer "k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "clustercomponent_defaulter"

type Reconciler struct {
	ctx        context.Context
	log        *zap.SugaredLogger
	client     ctrlruntimeclient.Client
	recorder   record.EventRecorder
	seedGetter provider.SeedGetter
	workerName string
}

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	seedGetter provider.SeedGetter,
	workerName string) error {

	reconciler := &Reconciler{
		ctx:        ctx,
		log:        log.Named(ControllerName),
		client:     mgr.GetClient(),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		seedGetter: seedGetter,
		workerName: workerName,
	}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}
	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.client.Get(r.ctx, request.NamespacedName, cluster); err != nil {
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
		kubermaticv1.ClusterConditionComponentDefaulterReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return nil, r.reconcile(log, cluster)
		},
	)
	if err != nil {
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
		r.log.With("error", err).With("cluster", request.NamespacedName.Name).Error("failed to reconcile cluster")
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) error {
	log.Debug("Syncing cluster")

	s, err := r.seedGetter()
	if err != nil {
		return fmt.Errorf("Error occurred while getting seed: %w", err)
	}
	defaults := kubermaticv1.ComponentSettings{
		Apiserver: kubermaticv1.APIServerSettings{
			DeploymentSettings:          kubermaticv1.DeploymentSettings{Replicas: utilpointer.Int32Ptr(int32(2))},
			EndpointReconcilingDisabled: utilpointer.BoolPtr(true),
		},
		ControllerManager: kubermaticv1.DeploymentSettings{
			Replicas: utilpointer.Int32Ptr(int32(1))},
		Scheduler: kubermaticv1.DeploymentSettings{
			Replicas: utilpointer.Int32Ptr(int32(1))},
	}
	// Override hardcoded defaults with default settings in Seed
	if s.Spec.ComponentSettings != nil {
		if s, ok := s.Spec.ComponentSettings["default"]; ok {
			defaults = s
		}
	}
	// TODO(irozzo) The following logic is probably wrong, defaulting should be
	// done at admission and not here, because it should be edge-driven and not
	// level-driven, especially when default can change dynamically.
	targetComponentsOverride := cluster.Spec.ComponentsOverride.DeepCopy()
	if targetComponentsOverride.Apiserver.Replicas == nil {
		targetComponentsOverride.Apiserver.Replicas = defaults.Apiserver.Replicas
	}
	if targetComponentsOverride.Apiserver.Resources == nil {
		targetComponentsOverride.Apiserver.Resources = defaults.Apiserver.Resources
	}
	if targetComponentsOverride.Apiserver.EndpointReconcilingDisabled == nil {
		targetComponentsOverride.Apiserver.EndpointReconcilingDisabled = defaults.Apiserver.EndpointReconcilingDisabled
	}
	if targetComponentsOverride.ControllerManager.Replicas == nil {
		targetComponentsOverride.ControllerManager.Replicas = defaults.ControllerManager.Replicas
	}
	if targetComponentsOverride.ControllerManager.Resources == nil {
		targetComponentsOverride.ControllerManager.Resources = defaults.ControllerManager.Resources
	}
	if targetComponentsOverride.Scheduler.Replicas == nil {
		targetComponentsOverride.Scheduler.Replicas = defaults.Scheduler.Replicas
	}
	if targetComponentsOverride.Scheduler.Resources == nil {
		targetComponentsOverride.Scheduler.Resources = defaults.Scheduler.Resources
	}
	if targetComponentsOverride.Etcd.Resources == nil {
		targetComponentsOverride.Etcd.Resources = defaults.Etcd.Resources
	}
	if targetComponentsOverride.Prometheus.Resources == nil {
		targetComponentsOverride.Prometheus.Resources = defaults.Prometheus.Resources
	}

	if apiequality.Semantic.DeepEqual(&cluster.Spec.ComponentsOverride, targetComponentsOverride) {
		return nil
	}

	oldCluster := cluster.DeepCopy()
	targetComponentsOverride.DeepCopyInto(&cluster.Spec.ComponentsOverride)
	if err := r.client.Patch(r.ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to update componentsOverride: %v", err)
	}
	log.Info("Successfully defaulted componentsOverride")
	return nil
}
