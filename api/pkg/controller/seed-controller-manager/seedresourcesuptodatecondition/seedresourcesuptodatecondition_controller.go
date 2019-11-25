package seedresourcesuptodatecondition

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1helper "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1/helper"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "seed_resources_up_to_date_condition_controller"

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	workerName string,
) error {
	r := &reconciler{
		ctx:        ctx,
		log:        log.Named(ControllerName),
		client:     mgr.GetClient(),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		workerName: workerName,
	}

	ctrlOptions := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	typesToWatch := []runtime.Object{
		&appsv1.Deployment{},
		&appsv1.StatefulSet{},
	}
	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())); err != nil {
			return fmt.Errorf("failed to create watch for %T: %v", t, err)
		}
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{})
}

type reconciler struct {
	ctx        context.Context
	log        *zap.SugaredLogger
	client     ctrlruntimeclient.Client
	recorder   record.EventRecorder
	workerName string
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := r.client.Get(r.ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, fmt.Errorf("failed to get cluster %q: %v", request.Name, err)
	}

	// Add a wrapping here so we can emit an event on error
	err := r.reconcile(cluster)
	if err != nil {
		r.log.With("cluster", request.Name).Errorw("Failed to reconcile cluster", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(cluster *kubermaticv1.Cluster) error {
	if r.workerName != cluster.Labels[kubermaticv1.WorkerNameLabelKey] {
		return nil
	}

	if cluster.Spec.Pause {
		return nil
	}

	upToDate, err := r.seedResourcesUpToDate(cluster)
	if err != nil {
		return err
	}

	oldCluster := cluster.DeepCopy()
	if !upToDate {
		kubermaticv1helper.SetClusterCondition(
			cluster,
			kubermaticv1.ClusterConditionSeedResourcesUpToDate,
			corev1.ConditionFalse,
			kubermaticv1.ReasonClusterUpdateSuccessful,
			"Some control plane components did not finish updating",
		)
		return r.client.Patch(r.ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
	}

	kubermaticv1helper.SetClusterCondition(
		cluster,
		kubermaticv1.ClusterConditionSeedResourcesUpToDate,
		corev1.ConditionTrue,
		kubermaticv1.ReasonClusterUpdateSuccessful,
		"All control plane components are up to date",
	)
	return r.client.Patch(r.ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster))
}

func (r *reconciler) seedResourcesUpToDate(cluster *kubermaticv1.Cluster) (bool, error) {

	listOpts := &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}

	statefulSets := &appsv1.StatefulSetList{}
	if err := r.client.List(r.ctx, statefulSets, listOpts); err != nil {
		return false, fmt.Errorf("failed to list statefulSets: %v", err)
	}
	for _, statefulSet := range statefulSets.Items {
		if statefulSet.Spec.Replicas == nil {
			return false, nil
		}
		if *statefulSet.Spec.Replicas != statefulSet.Status.UpdatedReplicas ||
			*statefulSet.Spec.Replicas != statefulSet.Status.CurrentReplicas ||
			*statefulSet.Spec.Replicas != statefulSet.Status.ReadyReplicas {
			return false, nil
		}
	}

	deployments := &appsv1.DeploymentList{}
	if err := r.client.List(r.ctx, deployments, listOpts); err != nil {
		return false, fmt.Errorf("failed to list deployments: %v", err)
	}

	for _, deployment := range deployments.Items {
		if deployment.Spec.Replicas == nil {
			return false, nil
		}
		if *deployment.Spec.Replicas != deployment.Status.UpdatedReplicas ||
			*deployment.Spec.Replicas != deployment.Status.AvailableReplicas ||
			*deployment.Spec.Replicas != deployment.Status.ReadyReplicas {
			return false, nil
		}
	}

	return true, nil
}
