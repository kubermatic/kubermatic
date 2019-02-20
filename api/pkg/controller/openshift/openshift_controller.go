package openshift

import (
	"context"
	"fmt"
	"time"

	clustercontroller "github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	openshiftresources "github.com/kubermatic/kubermatic/api/pkg/controller/openshift/resources"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"github.com/golang/glog"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kubermatic_openshift_controller"

// Check if the Reconciler fullfills the interface
// at compile time
var _ reconcile.Reconciler = &Reconciler{}

type Reconciler struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

func Add(mgr manager.Manager, numWorkers int, cloudProvider map[string]provider.CloudProvider, clusterPredicates predicate.Predicate) error {
	reconciler := &Reconciler{Client: mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetRecorder(ControllerName)}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}
	//TODO: Ensure only openshift clusters are handled via a predicate
	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}, clusterPredicates)
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Add a wrapping here so we can emit an event on error
	result, err := r.reconcile(ctx, cluster)
	if err != nil {
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.Spec.Pause {
		glog.V(6).Infof("skipping paused cluster %s", cluster.Name)
		return nil, nil
	}

	// Wait for namespace
	if cluster.Status.NamespaceName == "" {
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}
	ns := corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: cluster.Status.NamespaceName}, ns); err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, err
		}
		return &reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return nil, nil
}

func (r *Reconciler) deployments(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	deploymentName, deploymentCreator := openshiftresources.APIDeploymentCreator(
		&openshiftData{cluster: cluster, client: r.GetClient()})
	deployment := &appsv1.Deployment{}
	if err := r.Client.Get(ctx, nn(cluster.Status.NamespaceName, deploymentName), deployment); err != nil {
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to get deployment %s: %v", deploymentName, err)
		}
		deployment, err := deploymentCreator(&appsv1.Deployment)
		if err != nil {
			return fmt.Errorf("failed to get initial deployment %s from creator: %v", deploymentName, err)
		}
		if err := r.Create(ctx, deployment); err != nil {
			return fmt.Errorf("failed to create initial deployment %s: %v", deploymentName, err)
		}
	}
	deployment, err := deploymentCreator(deployment)
	if err != nil {
		return fmt.Errorf("failed to get deployment %s: %v", deploymentName, err)
	}
	if err := r.Create(ctx, deployment); err != nil {
		return fmt.Errorf("failed to create deployment %s: %v", deploymentName, err)
	}
	return nil
}

func (r *Reconciler) getOwnerRefForCluster(c *kubermaticv1.Cluster) metav1.OwnerReference {
	gv := kubermaticv1.SchemeGroupVersion
	return *metav1.NewControllerRef(c, gv.WithKind("Cluster"))
}

func (r *Reconciler) updateCluster(name string, modify func(*kubermaticv1.Cluster)) (updatedCluster *kubermaticv1.Cluster, err error) {
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cluster := &kubermaticv1.Cluster{}
		if err := r.Get(context.Background(), types.NamespacedName{Name: name}, cluster); err != nil {
			return err
		}
		modify(cluster)
		err := r.Update(context.Background(), cluster)
		if err == nil {
			updatedCluster = cluster
		}
		return err
	})

	return updatedCluster, err
}

// A cheap helper because I am too lazy to type this everytime
func nn(namespace, name string) types.NamespacedName {
	return types.NamespacedName{Namespace: namespace, Name: name}
}
