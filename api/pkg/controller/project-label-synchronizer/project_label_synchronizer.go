package projectlabelsynchronizer

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kubermatic_project_label_synchronizer"

type reconciler struct {
	ctx          context.Context
	log          *zap.SugaredLogger
	workerName   string
	masterClient ctrlruntimeclient.Client
	seedClients  map[string]ctrlruntimeclient.Client
}

// requestForSeed creates a reconcile.Request for each object, putting
// the seed name into the Namespace field and the objects Name into the
// name field.
func requestForSeed(seedName string) *handler.EnqueueRequestsFromMapFunc {
	toRequestFunc := handler.ToRequestsFunc(func(mo handler.MapObject) []reconcile.Request {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{Namespace: seedName, Name: mo.Meta.GetName()},
		}}
	})
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: toRequestFunc}
}

// requestsForProjectClusters creates a reconcile.Request for all clusters
// in all seeds that have a matching project-id label.
func requestsForProjectClusters(
	ctx context.Context,
	log *zap.SugaredLogger,
	seedClients map[string]ctrlruntimeclient.Client) *handler.EnqueueRequestsFromMapFunc {
	toRequestFunc := handler.ToRequestsFunc(func(mo handler.MapObject) []reconcile.Request {
		project, ok := mo.Object.(*kubermaticv1.Project)
		if !ok {
			err := fmt.Errorf("enqueueClustersForProject got an object that was not a *Project but a %T", mo.Object)
			// TODO: Add a zap logger to utilruntimes error handlers.
			utilruntime.HandleError(err)
			log.Error(err)
			return nil
		}
		log := log.With("project-name", project.Name)
		var requests []reconcile.Request
		for seedName, seedClient := range seedClients {
			clusters := &kubermaticv1.ClusterList{}
			// TODO: Worker-Name label
			if err := seedClient.List(ctx, &ctrlruntimeclient.ListOptions{}, clusters); err != nil {
				err := fmt.Errorf("failed to list clusters in seed %q: %v", seedName, err)
				utilruntime.HandleError(err)
				log.Error(err)
				continue
			}
			for _, cluster := range clusters.Items {
				if cluster.Labels[kubermaticv1.ProjectIDLabelKey] != project.Name {
					continue
				}
				requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
					Namespace: seedName,
					Name:      cluster.Name,
				}})
				log.Debugw("Created reconcile.Request", "cluster", cluster.Name, "seed", seedName)
			}
		}
		log.Debugw("Returning reconcile requests", "num-requests", len(requests))
		return requests
	})

	return &handler.EnqueueRequestsFromMapFunc{ToRequests: toRequestFunc}
}

func Add(
	ctx context.Context,
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerName string) error {

	log = log.Named(ControllerName)
	r := &reconciler{
		ctx:          ctx,
		log:          log,
		workerName:   workerName,
		masterClient: masterManager.GetClient(),
		seedClients:  map[string]ctrlruntimeclient.Client{},
	}

	ctrlOpts := controller.Options{
		Reconciler:              r,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, masterManager, ctrlOpts)
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	for seedName, seedManager := range seedManagers {
		r.seedClients[seedName] = seedManager.GetClient()

		seedClusterWatch := &source.Kind{Type: &kubermaticv1.Cluster{}}
		if err := seedClusterWatch.InjectCache(seedManager.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache for seed %q into watch: %v", seedName, err)
		}
		if err := c.Watch(seedClusterWatch, requestForSeed(seedName)); err != nil {
			return fmt.Errorf("failed to watch clusters in seed %q: %v", seedName, err)
		}
	}
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, requestsForProjectClusters(ctx, log, r.seedClients)); err != nil {
		return fmt.Errorf("failed to watch projects: %v", err)
	}
	return nil
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	seedName, clusterName := request.Namespace, request.Name
	log := r.log.With("seed", seedName, "cluster", clusterName)
	log.Debug("Processing")

	err := r.reconcile(log, seedName, clusterName)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(log *zap.SugaredLogger, seedName, clusterName string) error {

	client, ok := r.seedClients[seedName]
	if !ok {
		return errors.New("no client available for seed")
	}

	cluster := &kubermaticv1.Cluster{}
	if err := client.Get(r.ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Cluster not found, returning.")
			return nil
		}
		return fmt.Errorf("failed to get cluster: %v", err)
	}

	if val, _ := cluster.Labels[kubermaticv1.WorkerNameLabelKey]; val != r.workerName {
		log.Debug("Ignoring cluster because it has a different worker-name", "cluster-worker-name", val)
		return nil
	}

	projectName, hasProjectLabel := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if !hasProjectLabel {
		log.Debugf("Ignoring cluster because it doesn't have the project label %q", kubermaticv1.ProjectIDLabelKey)
		return nil
	}
	log = log.With(kubermaticv1.ProjectIDLabelKey, projectName)

	project := &kubermaticv1.Project{}
	if err := r.masterClient.Get(r.ctx, types.NamespacedName{Name: projectName}, project); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Project not found, returning.")
			return nil
		}
		return fmt.Errorf("failed to get project %q: %v", projectName, err)
	}

	if len(project.Labels) == 0 {
		log.Debug("Project has no labels, nothing to do")
		return nil
	}

	labelsToApply := map[string]string{}
	for key, value := range project.Labels {
		labelsToApply[key] = value
	}

	if err := r.updateCluster(cluster.Name, client, func(c *kubermaticv1.Cluster) {
		if c.Labels == nil {
			c.Labels = map[string]string{}
		}
		for key, value := range labelsToApply {
			c.Labels[key] = value
		}
	}); err != nil {
		return fmt.Errorf("failed to update cluster: %v", err)
	}

	return nil
}

func (r *reconciler) updateCluster(name string, client ctrlruntimeclient.Client, modify func(*kubermaticv1.Cluster)) error {
	cluster := &kubermaticv1.Cluster{}
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		if err := client.Get(r.ctx, types.NamespacedName{Name: name}, cluster); err != nil {
			return err
		}
		modify(cluster)
		return client.Update(r.ctx, cluster)
	})
}
