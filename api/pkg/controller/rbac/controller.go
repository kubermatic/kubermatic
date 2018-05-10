package rbac

import (
	"errors"
	"time"

	"github.com/golang/glog"

	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
)

// Controller stores necessary components that are required to implement RBACGenerator
type Controller struct {
	queue workqueue.RateLimitingInterface
}

// New creates a new RBACGenerator controller that is responsible for
// managing RBAC roles for project's resources
func New() (*Controller, error) {
	return &Controller{
		queue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "RBACGenerator"),
	}, nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (cc *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	glog.Infof("Starting RBACGenerator controller with %d workers", workerCount)
	defer glog.Info("Shutting down RBACGenerator controller")

	for i := 0; i < workerCount; i++ {
		go wait.Until(cc.runWorker, time.Second, stopCh)
	}

	<-stopCh
}

func (cc *Controller) runWorker() {
	for cc.processNextItem() {
	}
}

func (cc *Controller) processNextItem() bool {
	key, quit := cc.queue.Get()
	if quit {
		return false
	}
	defer cc.queue.Done(key)

	err := cc.sync(key.(string))

	cc.handleErr(err, key)
	return true
}

func (cc *Controller) sync(key string) error {
	return errors.New("not implemented")
}

// handleErr checks if an error happened and makes sure we will retry later.
func (cc *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		cc.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if cc.queue.NumRequeues(key) < 5 {
		glog.V(0).Infof("Error syncing %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		cc.queue.AddRateLimited(key)
		return
	}

	cc.queue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.V(0).Infof("Dropping %q out of the queue: %v", key, err)
}
