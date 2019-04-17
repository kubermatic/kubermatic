package seedproxy

import (
	"context"

	glog "github.com/kubermatic/glog-logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler stores all components required for monitoring
type Reconciler struct {
	ctrlruntimeclient.Client
	userClusterConnProvider userClusterConnectionProvider

	recorder record.EventRecorder
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	glog.V(4).Info("Reconciling seed proxies...")

	kubeconfig := &corev1.Secret{}
	if err := r.Get(ctx, ctrlruntimeclient.ObjectKey{Namespace: "kubermatic", Name: "kubeconfig"}, kubeconfig); err != nil {
		if errors.IsNotFound(err) {
			glog.V(6).Info("No kubermatic/kubeconfig secret found.")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	config := &clientcmdapi.Config{}

	return reconcile.Result{}, nil

	// cluster := &kubermaticv1.Cluster{}
	// if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
	// 	if kubeapierrors.IsNotFound(err) {
	// 		return reconcile.Result{}, nil
	// 	}
	// 	return reconcile.Result{}, err
	// }

	// if cluster.Spec.Pause {
	// 	glog.V(4).Infof("skipping cluster %s due to it was set to paused", cluster.Name)
	// 	return reconcile.Result{}, nil
	// }

	// if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
	// 	return reconcile.Result{}, nil
	// }

	// if cluster.DeletionTimestamp != nil {
	// 	// Cluster got deleted - all monitoring components will be automatically garbage collected (Due to the ownerRef)
	// 	return reconcile.Result{}, nil
	// }

	// if cluster.Status.NamespaceName == "" {
	// 	glog.V(4).Infof("skipping cluster %s because it has no namespace yet", cluster.Name)
	// 	return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	// }

	// // Wait until the UCCM is ready - otherwise we deploy with missing RBAC resources
	// if !cluster.Status.Health.UserClusterControllerManager {
	// 	glog.V(6).Infof("skipping cluster %s because the UserClusterControllerManager is not ready yet", cluster.Name)
	// 	return reconcile.Result{RequeueAfter: healthCheckPeriod}, nil
	// }

	// // Add a wrapping here so we can emit an event on error
	// result, err := r.reconcile(ctx, cluster)
	// if err != nil {
	// 	glog.Errorf("Failed to reconcile cluster %s: %v", cluster.Name, err)
	// 	r.recorder.Eventf(cluster, corev1.EventTypeWarning, "ReconcilingError", "%v", err)
	// }
	// if result == nil {
	// 	result = &reconcile.Result{}
	// }
	// return *result, err
}

// func (r *Reconciler) reconcile(ctx context.Context, cluster *kubermaticv1.Cluster) (*reconcile.Result, error) {
// 	glog.V(4).Infof("Reconciling cluster %s", cluster.Name)

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	data, err := r.getClusterTemplateData(context.Background(), r.Client, cluster)
// 	if err != nil {
// 		return nil, err
// 	}

// 	// check that all service accounts are created
// 	if err := r.ensureServiceAccounts(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all roles are created
// 	if err := r.ensureRoles(ctx, cluster); err != nil {
// 		return nil, err
// 	}

// 	// check that all role bindings are created
// 	if err := r.ensureRoleBindings(ctx, cluster); err != nil {
// 		return nil, err
// 	}

// 	// check that all secrets are created
// 	if err := r.ensureSecrets(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all ConfigMaps are available
// 	if err := r.ensureConfigMaps(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all Deployments are available
// 	if err := r.ensureDeployments(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all StatefulSets are created
// 	if err := r.ensureStatefulSets(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	// check that all VerticalPodAutoscaler's are created
// 	if err := r.ensureVerticalPodAutoscalers(ctx, cluster); err != nil {
// 		return nil, err
// 	}

// 	// check that all Services's are created
// 	if err := r.ensureServices(ctx, cluster, data); err != nil {
// 		return nil, err
// 	}

// 	return &reconcile.Result{}, nil
// }
