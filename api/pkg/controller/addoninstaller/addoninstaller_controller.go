package addoninstaller

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

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
	workerName       string
	kubernetesAddons []string
	openshiftAddons  []string
	ctrlruntimeclient.Client
	recorder record.EventRecorder
}

func Add(
	log *zap.SugaredLogger,
	mgr manager.Manager,
	numWorkers int,
	workerName string,
	kubernetesAddons,
	openshiftAddons []string) error {
	log = log.Named(ControllerName)

	reconciler := &Reconciler{
		log:              log,
		workerName:       workerName,
		kubernetesAddons: kubernetesAddons,
		openshiftAddons:  openshiftAddons,
		Client:           mgr.GetClient(),
		recorder:         mgr.GetRecorder(ControllerName),
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
		if err := mgr.GetClient().List(context.Background(), &ctrlruntimeclient.ListOptions{}, clusterList); err != nil {
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
	result, err := r.reconcile(ctx, log, cluster)
	if result == nil {
		result = &reconcile.Result{}
	}
	if err != nil {
		log.Errorw("Reconciling failed", zap.Error(err))
		r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	}
	return *result, err
}

func (r *Reconciler) reconcile(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
	// This controller handles Kubernetes & OpenShift cluster.
	// Based on the type we install different default addons
	var addonsToInstall []string
	if cluster.Annotations["kubermatic.io/openshift"] != "" {
		log = log.With("clustertype", "openshift")
		addonsToInstall = r.openshiftAddons
	} else {
		log = log.With("clustertype", "kubernetes")
		addonsToInstall = r.kubernetesAddons
	}

	if cluster.Spec.Pause {
		log.Debug("Skipping because the cluster is paused")
		return nil, nil
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		log.Debug("Skipping because the cluster has a different worker name set")
		return nil, nil
	}

	// Wait until the Apiserver is running to ensure the namespace exists at least.
	// Just checking for cluster.status.namespaceName is not enough as it gets set before the namespace exists
	if kubermaticv1.HealthStatusDown == cluster.Status.ExtendedHealth.Apiserver {
		log.Debug("Skipping because the API server is not running")
		return &reconcile.Result{RequeueAfter: 1 * time.Second}, nil
	}

	return nil, r.ensureAddons(ctx, log, cluster, addonsToInstall)
}

func (r *Reconciler) ensureAddons(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, addons []string) error {
	for _, addonName := range addons {
		addon := &kubermaticv1.Addon{}
		name := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: addonName}
		addonLog := log.With("addon", name)
		err := r.Get(ctx, name, addon)
		if err == nil {
			addonLog.Debug("Addon already exists")
			continue
		}
		if !kerrors.IsNotFound(err) {
			return fmt.Errorf("failed to get addon %q: %v", addonName, err)
		}
		if err := r.createAddon(ctx, addonLog, addonName, cluster); err != nil {
			return fmt.Errorf("failed to create addon %q: %v", addonName, err)
		}
	}
	return nil
}

func (r *Reconciler) createAddon(ctx context.Context, log *zap.SugaredLogger, addonName string, cluster *kubermaticv1.Cluster) error {
	gv := kubermaticv1.SchemeGroupVersion

	addon := &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:            addonName,
			Namespace:       cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))},
			Labels:          map[string]string{},
		},
		Spec: kubermaticv1.AddonSpec{
			Name: addonName,
			Cluster: corev1.ObjectReference{
				Name:       cluster.Name,
				Namespace:  "",
				UID:        cluster.UID,
				APIVersion: cluster.APIVersion,
				Kind:       "Cluster",
			},
		},
	}

	if r.workerName != "" {
		addon.Labels[kubermaticv1.WorkerNameLabelKey] = r.workerName
	}

	if err := r.Create(ctx, addon); err != nil {
		return fmt.Errorf("failed to create addon %q: %v", addonName, err)
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
