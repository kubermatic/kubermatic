package usersshkeysclustersynchronizer

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	controllerutil "github.com/kubermatic/kubermatic/api/pkg/controller/util"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	kubeapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	ControllerName = "usersshkeys_cluster_synchronizer"
)

// Reconciler is a controller which is responsible for managing clusters
type Reconciler struct {
	ctx         context.Context
	client      ctrlruntimeclient.Client
	log         *zap.SugaredLogger
	workerName  string
	seedClients map[string]ctrlruntimeclient.Client
}

func Add(
	ctx context.Context,
	mgr manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	workerName string,
	numWorkers int,
) error {

	reconciler := &Reconciler{
		ctx:         ctx,
		log:         log.Named(ControllerName),
		workerName:  workerName,
		client:      mgr.GetClient(),
		seedClients: map[string]ctrlruntimeclient.Client{},
	}

	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers})
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	for seedName, seedManager := range seedManagers {
		reconciler.seedClients[seedName] = seedManager.GetClient()

		clusterSource := &source.Kind{Type: &kubermaticv1.Cluster{}}
		if err := clusterSource.InjectCache(mgr.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache into clusterSource for seed %s: %v", seedName, err)
		}
		if err := c.Watch(
			clusterSource,
			controllerutil.EnqueueClusterScopedObjectWithSeedName(seedName),
		); err != nil {
			return fmt.Errorf("failed to establish watch for clusters in seed %s: %v", seedName, err)
		}
	}

	if err := c.Watch(
		&source.Kind{Type: &kubermaticv1.UserSSHKey{}},
		enqueueAllClusters(reconciler.seedClients),
	); err != nil {
		return fmt.Errorf("failed to create watch for userSSHKey: %v", err)
	}

	return nil
}

func (r *Reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With("request", request)
	log.Debug("Processing")

	err := r.reconcile(log, request)
	if err != nil {
		log.Errorw("Reconciliation failed", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (r *Reconciler) reconcile(log *zap.SugaredLogger, request reconcile.Request) error {
	seedClient, ok := r.seedClients[request.Namespace]
	if !ok {
		log.Errorw("Got request for seed we don't have a client for", "seed", request.Namespace)
		// The clients are inserted during controller initialzation, so there is no point in retrying
		return nil
	}

	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(r.ctx, types.NamespacedName{Name: request.Name}, cluster); err != nil {
		if kubeapierrors.IsNotFound(err) {
			log.Debug("Could not find cluster")
			return nil
		}
		return fmt.Errorf("failed to get cluster %s from seed %s: %v", cluster.Name, request.Namespace, err)
	}

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != r.workerName {
		log.Debugw(
			"Skipping because the cluster has a different worker name set",
			"cluster-worker-name", cluster.Labels[kubermaticv1.WorkerNameLabelKey],
		)
		return nil
	}

	if cluster.Spec.Pause {
		log.Debug("Skipping cluster reconciling because it was set to paused")
		return nil
	}

	if cluster.DeletionTimestamp == nil {
		return nil
	}

	userSSHKeys := &kubermaticv1.UserSSHKeyList{}
	if err := r.client.List(r.ctx, userSSHKeys); err != nil {
		return fmt.Errorf("failed to list userSSHKeys: %v", err)
	}

	for _, userSSHKey := range userSSHKeys.Items {
		userSSHKey.RemoveFromCluster(cluster.Name)
		if err := r.client.Update(r.ctx, &userSSHKey); err != nil {
			return fmt.Errorf("failed updating usersshkeys object: %v", err)

		}
	}
	return nil
}

// enqueueAllClusters enqueues all clusters
func enqueueAllClusters(clients map[string]ctrlruntimeclient.Client) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		var requests []reconcile.Request

		for seedName, client := range clients {
			clusterList := &kubermaticv1.ClusterList{}
			if err := client.List(context.Background(), clusterList); err != nil {
				utilruntime.HandleError(fmt.Errorf("failed to list Clusters in seed %s: %v", seedName, err))
				continue
			}
			for _, cluster := range clusterList.Items {
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Namespace: seedName,
					Name:      cluster.Name,
				}})
			}
		}

		return requests
	})}
}
