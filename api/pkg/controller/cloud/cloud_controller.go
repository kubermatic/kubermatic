package cloud

import (
	"context"
	"fmt"
	"time"

	clustercontroller "github.com/kubermatic/kubermatic/api/pkg/controller/cluster"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

const ControllerName = "kubermatic_cloud_controller"

// Check if the Reconciler fullfills the interface
// at compile time
var _ reconcile.Reconciler = &Reconciler{}

type Reconciler struct {
	client.Client
	scheme        *runtime.Scheme
	recorder      record.EventRecorder
	cloudProvider map[string]provider.CloudProvider
}

func Add(mgr manager.Manager, numWorkers int, cloudProvider map[string]provider.CloudProvider, clusterPredicates predicate.Predicate) error {
	reconciler := &Reconciler{Client: mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		recorder:      mgr.GetRecorder(ControllerName),
		cloudProvider: cloudProvider}

	c, err := controller.New(ControllerName, mgr,
		controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}
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
	if result == nil {
		result = &reconcile.Result{}
	}
	if err != nil {
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
		return *result, err
	}
	_, err = r.updateCluster(cluster.Name, func(c *kubermaticv1.Cluster) {
		c.Status.Health.CloudProviderInfrastructure = true
	})
	return *result, err
}

func (r *Reconciler) reconcile(_ context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	if cluster.Spec.Pause {
		glog.V(6).Infof("skipping paused cluster %s", cluster.Name)
		return nil, nil
	}
	_, prov, err := provider.ClusterCloudProvider(r.cloudProvider, cluster)
	if err != nil {
		return nil, err
	}
	if prov == nil {
		return nil, fmt.Errorf("no valid provider specified")
	}
	if cluster.DeletionTimestamp != nil {
		finalizers := sets.NewString(cluster.Finalizers...)
		if finalizers.Has(clustercontroller.InClusterLBCleanupFinalizer) ||
			finalizers.Has(clustercontroller.InClusterPVCleanupFinalizer) ||
			finalizers.Has(clustercontroller.NodeDeletionFinalizer) {
			return &reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}
		_, err = prov.CleanUpCloudProvider(cluster, r.updateCluster)
		return nil, err
	}

	_, err = prov.InitializeCloudProvider(cluster, r.updateCluster)
	return nil, err
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
