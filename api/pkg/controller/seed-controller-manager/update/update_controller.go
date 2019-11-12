package update

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1helper "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1/helper"
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	"github.com/kubermatic/kubermatic/api/pkg/version"
	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

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
	updateManager *version.Manager
	ctrlruntimeclient.Client
	recorder                      record.EventRecorder
	userClusterConnectionProvider client.UserClusterConnectionProvider
	log                           *zap.SugaredLogger
}

// Add creates a new update controller
func Add(mgr manager.Manager, numWorkers int, workerName string, updateManager *version.Manager,
	userClusterConnectionProvider client.UserClusterConnectionProvider, log *zap.SugaredLogger) error {
	reconciler := &Reconciler{
		workerName:                    workerName,
		updateManager:                 updateManager,
		Client:                        mgr.GetClient(),
		recorder:                      mgr.GetEventRecorderFor(ControllerName),
		userClusterConnectionProvider: userClusterConnectionProvider,
		log:                           log,
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
	result, err := kubermaticv1helper.ClusterReconcileWrapper(
		ctx,
		r.Client,
		r.workerName,
		cluster,
		kubermaticv1.ClusterConditionUpdateControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, cluster)
		},
	)
	if err != nil {
		r.log.Errorw("Failed to reconcile cluster", "namespace", request.NamespacedName.String(), zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {

	if !cluster.Status.ExtendedHealth.AllHealthy() {
		// Cluster not healthy yet. Nothing to do.
		// If it gets healthy we'll get notified by the event. No need to requeue
		return nil, nil
	}

	clusterType := v1.KubernetesClusterType
	if _, ok := cluster.Annotations["kubermatic.io/openshift"]; ok {
		clusterType = v1.OpenShiftClusterType
	}

	// NodeUpdate may need the controlplane to be updated first
	updated, err := r.controlPlaneUpgrade(ctx, cluster, clusterType)
	if err != nil {
		return nil, fmt.Errorf("failed to update the controlplane: %v", err)
	}
	// Give the controller time to do the update
	// TODO: This is not really safe. We should add a `Version` to the status
	// that gets incremented when the controller does this. Combined with a
	// `SeedResourcesUpToDate` condition, that should do the trick
	if updated {
		return &reconcile.Result{RequeueAfter: time.Minute}, nil
	}

	if err := r.nodeUpdate(ctx, cluster, clusterType); err != nil {
		return nil, fmt.Errorf("failed to update machineDeployments: %v", err)
	}

	return nil, nil
}

func (r *Reconciler) nodeUpdate(ctx context.Context, cluster *kubermaticv1.Cluster, clusterType string) error {
	c, err := r.userClusterConnectionProvider.GetClient(cluster)
	if err != nil {
		return fmt.Errorf("failed to get usercluster client: %v", err)
	}

	machineDeployments := &clusterv1alpha1.MachineDeploymentList{}
	// Kubermatic only creates MachineDeployments in the kube-system namespace, everything else is essentially unsupported
	if err := c.List(ctx, machineDeployments, ctrlruntimeclient.InNamespace("kube-system")); err != nil {
		return fmt.Errorf("failed to list MachineDeployments: %v", err)
	}

	for _, md := range machineDeployments.Items {
		targetVersion, err := r.updateManager.AutomaticNodeUpdate(md.Spec.Template.Spec.Versions.Kubelet, clusterType, cluster.Spec.Version.String())
		if err != nil {
			return fmt.Errorf("failed to get automatic update for machinedeployment %s/%s that has version %q: %v", md.Namespace, md.Name, md.Spec.Template.Spec.Versions.Kubelet, err)
		}
		if targetVersion == nil {
			continue
		}
		md.Spec.Template.Spec.Versions.Kubelet = targetVersion.Version.String()
		// DeepCopy it so we don't get a NPD when we return an error
		if err := c.Update(ctx, md.DeepCopy()); err != nil {
			return fmt.Errorf("failed to update MachineDeployment %s/%s to %q: %v", md.Namespace, md.Name, md.Spec.Template.Spec.Versions.Kubelet, err)
		}
		r.recorder.Eventf(cluster, corev1.EventTypeNormal, "AutoUpdateMachineDeployment", "Triggered automatic update of MachineDeployment %s/%s to version %q", md.Namespace, md.Name, targetVersion.Version.String())
	}

	return nil
}

func (r *Reconciler) controlPlaneUpgrade(ctx context.Context, cluster *kubermaticv1.Cluster, clusterType string) (upgraded bool, err error) {
	update, err := r.updateManager.AutomaticControlplaneUpdate(cluster.Spec.Version.String(), clusterType)
	if err != nil {
		return false, fmt.Errorf("failed to get automatic update for cluster for version %s: %v", cluster.Spec.Version.String(), err)
	}
	if update == nil {
		return false, nil
	}
	oldCluster := cluster.DeepCopy()

	cluster.Spec.Version = *semver.NewSemverOrDie(update.Version.String())
	// Invalidating the health to prevent automatic updates directly on the next processing.
	cluster.Status.ExtendedHealth.Apiserver = kubermaticv1.HealthStatusDown
	cluster.Status.ExtendedHealth.Controller = kubermaticv1.HealthStatusDown
	cluster.Status.ExtendedHealth.Scheduler = kubermaticv1.HealthStatusDown
	if err := r.Patch(ctx, cluster, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return false, fmt.Errorf("failed to update cluster: %v", err)
	}
	return true, nil
}
