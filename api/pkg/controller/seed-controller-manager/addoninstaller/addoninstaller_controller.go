package addoninstaller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticv1helper "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1/helper"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kubermatic_addoninstaller_controller"

type Reconciler struct {
	log              *zap.SugaredLogger
	kubernetesAddons kubermaticv1.AddonList
	openshiftAddons  kubermaticv1.AddonList
	workerName       string
	ctrlruntimeclient.Client
	recorder record.EventRecorder
}

func Add(
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	kubernetesAddons kubermaticv1.AddonList,
	openshiftAddons kubermaticv1.AddonList,
) error {
	log = log.Named(ControllerName)

	reconciler := &Reconciler{
		log:              log,
		workerName:       workerName,
		kubernetesAddons: kubernetesAddons,
		openshiftAddons:  openshiftAddons,
		Client:           mgr.GetClient(),
		recorder:         mgr.GetEventRecorderFor(ControllerName),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	})
	if err != nil {
		return fmt.Errorf("failed to create controller: %v", err)
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create watch for clusters: %v", err)
	}

	enqueueClusterForNamespacedObject := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		clusterList := &kubermaticv1.ClusterList{}
		if err := mgr.GetClient().List(context.Background(), clusterList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %v", err))
			log.Errorw("Failed to list clusters", zap.Error(err))
			return []reconcile.Request{}
		}
		for _, cluster := range clusterList.Items {
			if cluster.Status.NamespaceName == a.Meta.GetNamespace() {
				return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: cluster.Name}}}
			}
		}
		return []reconcile.Request{}
	})}
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Addon{}}, enqueueClusterForNamespacedObject); err != nil {
		return fmt.Errorf("failed to create watch for Addons: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Skipping because the cluster is already gone")
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
		kubermaticv1.ClusterConditionAddonInstallerControllerReconcilingSuccess,
		func() (*reconcile.Result, error) {
			return r.reconcile(ctx, log, cluster)
		},
	)
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Event(cluster, corev1.EventTypeWarning, "ReconcilingError", err.Error())
	}
	if result == nil {
		result = &reconcile.Result{}
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// This controller handles Kubernetes & OpenShift cluster.
	// Based on the type we install different default addons
	var addonsToInstall *kubermaticv1.AddonList
	if cluster.IsOpenshift() {
		log = log.With("clustertype", "openshift")
		addonsToInstall = r.openshiftAddons.DeepCopy()
	} else {
		log = log.With("clustertype", "kubernetes")
		addonsToInstall = r.kubernetesAddons.DeepCopy()
	}

	// Wait until the Apiserver is running to ensure the namespace exists at least.
	// Just checking for cluster.status.namespaceName is not enough as it gets set before the namespace exists
	if cluster.Status.ExtendedHealth.Apiserver != kubermaticv1.HealthStatusUp {
		log.Debug("Skipping because the API server is not running")
		return &reconcile.Result{RequeueAfter: 1 * time.Second}, nil
	}

	return nil, r.ensureAddons(ctx, log, cluster, *addonsToInstall)
}

func (r *Reconciler) ensureAddons(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, addons kubermaticv1.AddonList) error {
	for _, addon := range addons.Items {
		name := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: addon.Name}
		addonLog := log.With("addon", name)
		err := r.Get(ctx, name, &kubermaticv1.Addon{})
		if err == nil {
			addonLog.Debug("Addon already exists")
			continue
		}
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to get addon %q: %v", addon.Name, err)
		}
		if err := r.createAddon(ctx, addonLog, addon, cluster); err != nil {
			return fmt.Errorf("failed to create addon %q: %v", addon.Name, err)
		}
	}
	return nil
}

func (r *Reconciler) createAddon(ctx context.Context, log *zap.SugaredLogger, addon kubermaticv1.Addon, cluster *kubermaticv1.Cluster) error {
	gv := kubermaticv1.SchemeGroupVersion

	addon.Namespace = cluster.Status.NamespaceName
	addon.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))}
	if addon.Labels == nil {
		addon.Labels = map[string]string{}
	}
	addon.Spec.Name = addon.Name
	addon.Spec.Cluster = corev1.ObjectReference{
		Name:       cluster.Name,
		Namespace:  "",
		UID:        cluster.UID,
		APIVersion: cluster.APIVersion,
		Kind:       "Cluster",
	}
	addon.Spec.IsDefault = true

	// Swallow IsAlreadyExists, we have predictable names and our cache may not be
	// up to date, leading us to think the addon wasn't installed yet.
	if err := r.Create(ctx, &addon); err != nil && !kerrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create addon %q: %v", addon.Name, err)
	}

	log.Info("Addon successfully created")

	err := wait.Poll(10*time.Millisecond, 10*time.Second, func() (bool, error) {
		err := r.Get(ctx, types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: addon.Name}, &kubermaticv1.Addon{})
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed waiting for addon %s to exist in the lister", addon.Name)
	}

	return nil
}
