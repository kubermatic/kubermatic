package update

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/version"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers prometheus.Gauge
}

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

// Controller stores necessary components that are required to implement Update
type Controller struct {
	queue         workqueue.RateLimitingInterface
	metrics       *Metrics
	updateManager Manager
	workerName    string

	kubermaticClient kubermaticclientset.Interface
	clusterLister    kubermaticv1lister.ClusterLister
	clusterSynced    cache.InformerSynced
}

// Manager specifies a set of methods to find suitable update versions for clusters
type Manager interface {
	AutomaticUpdate(from string) (*version.MasterVersion, error)
}

// New creates a new Update controller that is responsible for
// managing automatic updates to clusters while following a pre defined update path
func New(
	metrics *Metrics,
	updateManager Manager,
	workerName string,
	kubermaticClient kubermaticclientset.Interface,
	clusterInformer kubermaticv1informers.ClusterInformer) (*Controller, error) {
	c := &Controller{
		queue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Update"),
		metrics:          metrics,
		workerName:       workerName,
		kubermaticClient: kubermaticClient,
		updateManager:    updateManager,
	}

	prometheus.MustRegister(metrics.Workers)

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueue(obj.(*kubermaticv1.Cluster))
		},
		UpdateFunc: func(old, cur interface{}) {
			c.enqueue(cur.(*kubermaticv1.Cluster))
		},
		DeleteFunc: func(obj interface{}) {
			cluster, ok := obj.(*kubermaticv1.Cluster)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					runtime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				cluster, ok = tombstone.Obj.(*kubermaticv1.Cluster)
				if !ok {
					runtime.HandleError(fmt.Errorf("tombstone contained object that is not a Cluster %#v", obj))
					return
				}
			}
			c.enqueue(cluster)
		},
	})
	c.clusterLister = clusterInformer.Lister()
	c.clusterSynced = clusterInformer.Informer().HasSynced

	return c, nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	glog.Infof("Starting Update controller with %d workers", workerCount)
	defer glog.Info("Shutting down Update controller")

	if !cache.WaitForCacheSync(stopCh, c.clusterSynced) {
		runtime.HandleError(errors.New("unable to sync caches for Update controller"))
		return
	}

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	c.metrics.Workers.Set(float64(workerCount))
	<-stopCh
}

func (c *Controller) runWorker() {
	for c.processNextItem() {
	}
}

func (c *Controller) processNextItem() bool {
	key, quit := c.queue.Get()
	if quit {
		return false
	}
	defer c.queue.Done(key)

	err := c.sync(key.(string))

	c.handleErr(err, key)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		glog.V(0).Infof("Error syncing %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.V(0).Infof("Dropping %q out of the queue: %v", key, err)
}

func (c *Controller) enqueue(cluster *kubermaticv1.Cluster) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(cluster)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", cluster, err))
		return
	}

	c.queue.Add(key)
}

func (c *Controller) sync(key string) error {
	clusterFromCache, err := c.clusterLister.Get(key)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("cluster '%s' in work queue no longer exists", key)
			return nil
		}
		return err
	}

	cluster := clusterFromCache.DeepCopy()

	if cluster.Labels[kubermaticv1.WorkerNameLabelKey] != c.workerName {
		glog.V(8).Infof("skipping cluster %s due to different worker assigned to it", key)
		return nil
	}

	if !cluster.Status.Health.AllHealthy() {
		// Cluster not healthy yet. Nothing to do.
		// If it gets healthy we'll get notified by the event. No need to requeue
		return nil
	}

	if err := c.ensureAutomaticUpdatesAreApplied(cluster); err != nil {
		return err
	}

	return err
}

func (c *Controller) ensureAutomaticUpdatesAreApplied(cluster *kubermaticv1.Cluster) error {
	update, err := c.updateManager.AutomaticUpdate(cluster.Spec.Version)
	if err != nil {
		return fmt.Errorf("failed to get automatic update for cluster for version %s: %v", cluster.Spec.Version, err)
	}
	if update == nil {
		return nil
	}

	cluster.Spec.Version = update.Version.String()
	// Invalidating the health to prevent automatic updates directly on the next processing.
	cluster.Status.Health.Apiserver = false
	cluster.Status.Health.Controller = false
	cluster.Status.Health.Scheduler = false
	_, err = c.kubermaticClient.KubermaticV1().Clusters().Update(cluster)
	return err
}
