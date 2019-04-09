package update

import (
	"context"
	"fmt"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

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

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers prometheus.Gauge
}

// NewMetrics creates a new Metrics
// with default values initialized, so metrics always show up
func NewMetrics() *Metrics {
	cm := &Metrics{
		Workers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "kubermatic",
			Subsystem: "update_controller",
			Name:      "workers",
			Help:      "The number of running Update controller workers",
		}),
	}

	cm.Workers.Set(0)
	return cm
}

const (
	ControllerName = "kubermatic_update_controller"
)

type Reconciler struct {
	workerName    string
	metrics       *Metrics
	updateManager Manager
	ctrlruntimeclient.Client
	recorder record.EventRecorder
}

// Manager specifies a set of methods to find suitable update versions for clusters
type Manager interface {
	AutomaticUpdate(from string) (*version.MasterVersion, error)
}

// Add creates a new update controller
func Add(mgr manager.Manager, numWorkers int, workerName string, metrics *Metrics, updateManager Manager) error {
	reconciler := &Reconciler{
		workerName:    workerName,
		metrics:       metrics,
		updateManager: updateManager,
		Client:        mgr.GetClient(),
		recorder:      mgr.GetRecorder(ControllerName),
	}

	if err := prometheus.Register(metrics.Workers); err != nil {
		return fmt.Errorf("failed to register worker metrics: %v", err)
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

	if !cluster.Status.Health.AllHealthy() {
		// Cluster not healthy yet. Nothing to do.
		// If it gets healthy we'll get notified by the event. No need to requeue
		return nil
	}

	update, err := r.updateManager.AutomaticUpdate(cluster.Spec.Version.String())
	if err != nil {
		return fmt.Errorf("failed to get automatic update for cluster for version %s: %v", cluster.Spec.Version.String(), err)
	}
	if update == nil {
		return nil
	}

	cluster.Spec.Version = *semver.NewSemverOrDie(update.Version.String())
	// Invalidating the health to prevent automatic updates directly on the next processing.
	cluster.Status.Health.Apiserver = false
	cluster.Status.Health.Controller = false
	cluster.Status.Health.Scheduler = false
	if err := r.Update(ctx, cluster); err != nil {
		return fmt.Errorf("failed to update cluster: %v", err)
	}
	return nil
}
