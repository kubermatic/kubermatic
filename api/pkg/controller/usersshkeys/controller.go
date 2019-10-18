package usersshkeys

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "kubermatic_usersshkeys_controller"
)

// Reconciler is a controller which is responsible for managing clusters
type Reconciler struct {
	ctrlruntimeclient.Client
	log        *zap.SugaredLogger
	workerName string
	recorder   record.EventRecorder
}

func Add(
	mgr manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {

	reconciler := &Reconciler{
		log:        log,
		workerName: workerName,
		Client:     mgr.GetClient(),
		recorder:   mgr.GetRecorder(ControllerName),
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &kubermaticv1.UserSSHKey{}}, enqueueAllClusters(mgr.GetClient())); err != nil {
		return fmt.Errorf("failed to create watcher for userSSHKey: %v", err)
	}

	if err := c.Watch(&source.Kind{Type: &corev1.Secret{}}, controllerutil.EnqueueClusterForNamespacedObject(mgr.GetClient())); err != nil {
		return fmt.Errorf("failed to create watcher for secrets: %v", err)
	}

	return c.Watch(&source.Kind{Type: &kubermaticv1.Cluster{}}, &handler.EnqueueRequestForObject{})
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	log := r.log.With("request", request)
	log.Debug("Processing")

	cluster := &kubermaticv1.Cluster{}
	if err := r.Get(ctx, request.NamespacedName, cluster); err != nil {
		if kubeapierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		log.Debugw(
			"Skipping because the cluster has a different worker name set",
			"cluster-worker-name", cluster.Labels[kubermaticv1.WorkerNameLabelKey],
		)
		return reconcile.Result{}, nil
	}

	if cluster.Spec.Pause {
		log.Debug("Skipping cluster reconciling because it was set to paused")
		return reconcile.Result{}, nil
	}

	if err := r.reconcileCluster(ctx, cluster); err != nil {
		log.Errorw("Failed reconciling clusters user ssh secrets", "cluster", cluster.Name, zap.Error(err))
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) reconcileCluster(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	userSSHKeys := &kubermaticv1.UserSSHKeyList{}
	if err := r.Client.List(ctx, &ctrlruntimeclient.ListOptions{}, userSSHKeys); err != nil {
		return err
	}

	keys := buildUserSSHKeysForCluster(cluster.Name, userSSHKeys)

	return reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretCreatorGetter{updateUserSSHKeysSecrets(keys)}, cluster.Status.NamespaceName, r.Client)
}

func buildUserSSHKeysForCluster(clusterName string, list *kubermaticv1.UserSSHKeyList) []kubermaticv1.UserSSHKey {
	var clusterKeys []kubermaticv1.UserSSHKey
	for _, item := range list.Items {
		for _, clusterID := range item.Spec.Clusters {
			if clusterName == clusterID {
				clusterKeys = append(clusterKeys, item)
			}
		}
	}

	return clusterKeys
}

// enqueueAllClusters enqueues all clusters
func enqueueAllClusters(client ctrlruntimeclient.Client) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		clustersRequests := []reconcile.Request{}
		clusterList := &kubermaticv1.ClusterList{}
		if err := client.List(context.Background(), &ctrlruntimeclient.ListOptions{}, clusterList); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list Clusters: %v", err))
			return clustersRequests
		}

		for _, cluster := range clusterList.Items {
			clustersRequests = append(clustersRequests, reconcile.Request{NamespacedName: types.NamespacedName{Name: cluster.Name}})
		}

		return clustersRequests
	})}
}

// updateUserSSHKeysSecrets creates a secret in the seed cluster from the user ssh keys.
func updateUserSSHKeysSecrets(list []kubermaticv1.UserSSHKey) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.UserSSHKeys, func(existing *corev1.Secret) (secret *corev1.Secret, e error) {
			existing.Data = map[string][]byte{}

			for _, key := range list {
				existing.Data[key.Name] = []byte(key.Spec.PublicKey)
			}

			existing.Type = corev1.SecretTypeOpaque

			return existing, nil
		}
	}
}
