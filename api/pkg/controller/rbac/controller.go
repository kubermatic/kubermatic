package rbac

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"github.com/go-kit/kit/metrics"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers metrics.Gauge
}

// Controller stores necessary components that are required to implement RBACGenerator
type Controller struct {
	queue   workqueue.RateLimitingInterface
	metrics Metrics

	kubermaticClient kubermaticclientset.Interface
	projectLister    kubermaticv1lister.ProjectLister
	projectSynced    cache.InformerSynced
	userLister       kubermaticv1lister.UserLister
}

// New creates a new RBACGenerator controller that is responsible for
// managing RBAC roles for project's resources
func New(
	metrics Metrics,
	kubermaticClient kubermaticclientset.Interface,
	projectInformer kubermaticv1informers.ProjectInformer,
	userLister kubermaticv1lister.UserLister) (*Controller, error) {
	c := &Controller{
		queue:            workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "RBACGenerator"),
		metrics:          metrics,
		kubermaticClient: kubermaticClient,
		userLister:       userLister,
	}

	projectInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueue(obj.(*kubermaticv1.Project))
		},
		UpdateFunc: func(old, cur interface{}) {
			c.enqueue(cur.(*kubermaticv1.Project))
		},
		DeleteFunc: func(obj interface{}) {
			project, ok := obj.(*kubermaticv1.Project)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					runtime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				project, ok = tombstone.Obj.(*kubermaticv1.Project)
				if !ok {
					runtime.HandleError(fmt.Errorf("tombstone contained object that is not a Project %#v", obj))
					return
				}
			}
			c.enqueue(project)
		},
	})
	c.projectLister = projectInformer.Lister()
	c.projectSynced = projectInformer.Informer().HasSynced

	return c, nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	glog.Infof("Starting RBACGenerator controller with %d workers", workerCount)
	defer glog.Info("Shutting down RBACGenerator controller")

	if !cache.WaitForCacheSync(stopCh, c.projectSynced) {
		runtime.HandleError(errors.New("Unable to sync caches for RBACGenerator controller"))
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

func (c *Controller) enqueue(project *kubermaticv1.Project) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(project)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", project, err))
		return
	}

	c.queue.Add(key)
}
