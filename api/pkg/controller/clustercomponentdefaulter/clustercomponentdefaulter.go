package clustercomponentdefaulter

import (
	"context"
	"fmt"

	"github.com/go-test/deep"
	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1helper "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1/helper"

	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
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
	defaults   kubermaticv1.ComponentSettings
	workerName string
}

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	defaults kubermaticv1.ComponentSettings,
	workerName string) error {

	reconciler := &Reconciler{
		ctx:        ctx,
		log:        log.Named(ControllerName),
		client:     mgr.GetClient(),
		recorder:   mgr.GetRecorder(ControllerName),
		defaults:   defaults,
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

	if _, err := r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
		targetComponentsOverride.DeepCopyInto(&c.Spec.ComponentsOverride)
	}); err != nil {
		return fmt.Errorf("failed to update componentsOverride: %v", err)
	}
	log.Info("Successfully defaulted componentsOverride")
	return nil
}

func (r *Reconciler) updateCluster(name string, modify func(*kubermaticv1.Cluster)) (*kubermaticv1.Cluster, error) {
	cluster := &kubermaticv1.Cluster{}
	var diff []string
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := r.client.Get(r.ctx, types.NamespacedName{Name: name}, cluster); err != nil {
			return err
		}
		old := cluster.DeepCopy()
		modify(cluster)
		diff = deep.Equal(old, cluster)
		return r.client.Update(r.ctx, cluster)
	})
	if err != nil {
		err = fmt.Errorf("err: %v, diff: %v", err, diff)
	}
	return cluster, err
}
