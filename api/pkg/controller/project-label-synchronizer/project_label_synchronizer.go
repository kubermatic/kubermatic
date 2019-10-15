package projectlabelsynchronizer

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const ControllerName = "kubermatic_project_label_synchronizer"

type reconciler struct {
	ctx                     context.Context
	log                     *zap.SugaredLogger
	masterClient            ctrlruntimeclient.Client
	seedClients             map[string]ctrlruntimeclient.Client
	seedCaches              map[string]cache.Cache
	workerNameLabelSelector labels.Selector
}

// requestFromCluster returns a reconcile.Request for the project the given
// cluster belongs to, if any.
func requestFromCluster(log *zap.SugaredLogger) *handler.EnqueueRequestsFromMapFunc {
	toRequestFunc := handler.ToRequestsFunc(func(mo handler.MapObject) []reconcile.Request {
		cluster, ok := mo.Object.(*kubermaticv1.Cluster)
		if !ok {
			err := fmt.Errorf("Object was not a cluster but a %T", mo.Object)
			log.Error(err)
			utilruntime.HandleError(err)
			return nil
		}
		labelValue, hasLabel := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
		if !hasLabel {
			log.Debugw("Cluster has no project label", "cluster", cluster.Name)
			return nil
		}
		log.Debugw("Returning reconcile request for project", kubermaticv1.ProjectIDLabelKey, labelValue)
		return []reconcile.Request{{NamespacedName: types.NamespacedName{Name: labelValue}}}
	})
	return &handler.EnqueueRequestsFromMapFunc{ToRequests: toRequestFunc}
}

func Add(
	ctx context.Context,
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	workerNameLabelSelector labels.Selector) error {

	log = log.Named(ControllerName)
	r := &reconciler{
		ctx:                     ctx,
		log:                     log,
		masterClient:            masterManager.GetClient(),
		seedClients:             map[string]ctrlruntimeclient.Client{},
		seedCaches:              map[string]cache.Cache{},
		workerNameLabelSelector: workerNameLabelSelector,
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
		r.seedCaches[seedName] = seedManager.GetCache()

		seedClusterWatch := &source.Kind{Type: &kubermaticv1.Cluster{}}
		if err := seedClusterWatch.InjectCache(seedManager.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache for seed %q into watch: %v", seedName, err)
		}
		if err := c.Watch(seedClusterWatch, requestFromCluster(log)); err != nil {
			return fmt.Errorf("failed to watch clusters in seed %q: %v", seedName, err)
		}
	}
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to watch projects: %v", err)
	}
	return nil
}

func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := r.log.With(kubermaticv1.ProjectIDLabelKey, request.Name)
	log.Debug("Processing")

	err := r.reconcile(log, request)
	if err != nil {
		log.Errorw("ReconcilingError", zap.Error(err))
	}
	return reconcile.Result{}, err
}

func (r *reconciler) reconcile(log *zap.SugaredLogger, request reconcile.Request) error {
	project := &kubermaticv1.Project{}
	if err := r.masterClient.Get(r.ctx, request.NamespacedName, project); err != nil {
		if kerrors.IsNotFound(err) {
			log.Debug("Didn't find project, returning")
			return nil
		}
		return fmt.Errorf("failed to get project %s: %v", request.Name, err)
	}

	if len(project.Labels) == 0 {
		log.Debug("Project has no labels, nothing to do")
		return nil
	}

	// The main manager starts the seed managers, so we have to wait for them to be synced
	if err := r.waitForSeedCacheSync(); err != nil {
		return err
	}

	// We use an error aggregate to make sure we return an error if we encountered one but
	// still continue processing everything we can.
	var errs []error
	for seedName, seedClient := range r.seedClients {
		log := log.With("seed", seedName)

		listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: r.workerNameLabelSelector}
		unfilteredClusters := &kubermaticv1.ClusterList{}
		if err := seedClient.List(r.ctx, listOpts, unfilteredClusters); err != nil {
			errs = append(errs, fmt.Errorf("failed to list clusters in seed %q: %v", seedName, err))
			continue
		}

		filteredClusters := r.filterClustersByProjectID(log, project.Name, unfilteredClusters)
		for _, cluster := range filteredClusters {
			log := log.With("cluster", cluster.Name)
			changed, newClusterLabels := getLabelsForCluster(log, cluster.Labels, project.Labels)
			if !changed {
				log.Debug("Labels on cluster are already up to date")
				continue
			}
			log.Debug("Updating labels on cluster")
			if err := r.updateCluster(cluster.Name, seedClient, func(c *kubermaticv1.Cluster) {
				c.Labels = newClusterLabels
			}); err != nil {
				errs = append(errs, fmt.Errorf("failed to update cluster %q", cluster.Name))
			}
		}
	}

	return utilerrors.NewAggregate(errs)
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

// Wait for cache sync waits up to 30s for all seed caches to be synced.
func (r *reconciler) waitForSeedCacheSync() error {
	// TODO: Does this timeout actually stop the cache or just waiting for the cache?
	ctx, cancel := context.WithTimeout(r.ctx, 30*time.Second)
	defer cancel()
	for seedName, cache := range r.seedCaches {
		if success := cache.WaitForCacheSync(ctx.Done()); !success {
			return fmt.Errorf("failed waiting for cache of seed %q", seedName)
		}
	}

	return nil
}

func (r *reconciler) filterClustersByProjectID(
	log *zap.SugaredLogger,
	projectID string,
	clusters *kubermaticv1.ClusterList,
) []*kubermaticv1.Cluster {
	var result []*kubermaticv1.Cluster

	for idx, cluster := range clusters.Items {
		log := log.With("cluster", cluster.Name)
		if val := cluster.Labels[kubermaticv1.ProjectIDLabelKey]; val != projectID {
			log.Debugw("Ignoring cluster because it has the wrong project-id", "cluster-project-id", val)
			continue
		}

		result = append(result, &clusters.Items[idx])
	}

	return result
}

func getLabelsForCluster(
	log *zap.SugaredLogger,
	clusterLabels map[string]string,
	projectLabels map[string]string,
) (changed bool, newClusterLabels map[string]string) {
	// They shouldn't be nil as we skip projects without labels
	// and need a label on the cluster to associate it to a project
	// but better be safe than panicing.
	if clusterLabels == nil {
		clusterLabels = map[string]string{}
	}
	if projectLabels == nil {
		projectLabels = map[string]string{}
	}
	newClusterLabels = map[string]string{}

	for projectLabelKey, projectLabelValue := range projectLabels {
		if kubermaticv1.ProtectedClusterLabels.Has(projectLabelKey) {
			log.Info("Project wants to set protected label %q on cluster, skipping", projectLabelKey)
			continue
		}
		if clusterLabels[projectLabelKey] == projectLabelValue {
			log.Debugf("Label %q on cluster already has value of %q, nothing to do", projectLabelKey, projectLabelValue)
			continue
		}
		log.Debug("Setting label %q to value %q on cluster", projectLabelKey, projectLabelValue)
		clusterLabels[projectLabelKey] = projectLabelValue
		changed = true
	}
	return
}
