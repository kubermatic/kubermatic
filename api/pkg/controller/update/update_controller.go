package update

import (
	"context"
	"fmt"

	"github.com/golang/glog"

	"github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_update_controller"
)

type Reconciler struct {
	workerName    string
	updateManager Manager
	ctrlruntimeclient.Client
	recorder record.EventRecorder
}

// Manager specifies a set of methods to find suitable update versions for clusters
type Manager interface {
	AutomaticUpdate(from, clusterType string) (*version.MasterVersion, error)
}

// Add creates a new update controller
func Add(mgr manager.Manager, numWorkers int, workerName string, updateManager Manager) error {
	reconciler := &Reconciler{
		workerName:    workerName,
		updateManager: updateManager,
		Client:        mgr.GetClient(),
		recorder:      mgr.GetRecorder(ControllerName),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch: %v", err)
	}

	return nil
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
	err := r.reconcile(ctx, cluster)
	if err != nil {
		glog.Errorf("Failed to reconcile cluster %q: %v", request.NamespacedName.String(), err)
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		return nil
	}

	if !cluster.Status.ExtendedHealth.AllHealthy() {
		// Cluster not healthy yet. Nothing to do.
		// If it gets healthy we'll get notified by the event. No need to requeue
		return nil
	}

	clusterType := v1.KubernetesClusterType
	if _, ok := cluster.Annotations["kubermatic.io/openshift"]; ok {
		clusterType = v1.OpenShiftClusterType
	}

	update, err := r.updateManager.AutomaticUpdate(cluster.Spec.Version.String(), clusterType)
	if err != nil {
		return fmt.Errorf("failed to get automatic update for cluster for version %s: %v", cluster.Spec.Version.String(), err)
	}
	if update == nil {
		return nil
	}

	cluster.Spec.Version = *semver.NewSemverOrDie(update.Version.String())
	// Invalidating the health to prevent automatic updates directly on the next processing.
	cluster.Status.ExtendedHealth.Apiserver = kubermaticv1.HealthStatusDown
	cluster.Status.ExtendedHealth.Controller = kubermaticv1.HealthStatusDown
	cluster.Status.ExtendedHealth.Scheduler = kubermaticv1.HealthStatusDown
	if err := r.Update(ctx, cluster); err != nil {
		return fmt.Errorf("failed to update cluster: %v", err)
	}
	return nil
}
