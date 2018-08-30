package usercluster

import (
	"fmt"
	"time"

	"github.com/golang/glog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sinformersV1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	k8slistersV1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	// queueKey is the constant key added to the queue for deduplication.
	queueKey = "some user-cluster object"
)

// Controller controls objects in user-cluster
type Controller struct {
	client          kubernetes.Interface
	configMapLister k8slistersV1.ConfigMapLister
	queue           workqueue.RateLimitingInterface
}

// NewController creates a new controller for the specified data.
func NewController(client kubernetes.Interface,
	configMapInformer k8sinformersV1.ConfigMapInformer) (*Controller, error) {

	ctrl := &Controller{
		client:          client,
		configMapLister: configMapInformer.Lister(),
		queue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "configmaps"),
	}

	configMapInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(_ interface{}) { ctrl.queue.Add(queueKey) },
		DeleteFunc: func(_ interface{}) { ctrl.queue.Add(queueKey) },
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldConfigMap := oldObj.(*corev1.ConfigMap)
			newConfigMap := newObj.(*corev1.ConfigMap)
			if equality.Semantic.DeepEqual(oldConfigMap, newConfigMap) {
				return
			}
			ctrl.queue.Add(queueKey)
		},
	})
	return ctrl, nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed.
func (ctrl *Controller) Run(_ int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	go wait.Until(func() { ctrl.queue.Add(queueKey) }, time.Second*30, stopCh)
	go wait.Until(ctrl.runWorker, time.Second, stopCh)
	<-stopCh
}

// handleErr checks if an error happened and makes sure we will retry later.
func (ctrl *Controller) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		ctrl.queue.Forget(key)
		return
	}

	glog.V(0).Infof("Error syncing %v: %v", key, err)

	// Re-enqueue the key rate limited. Based on the rate limiter on the
	// queue and the re-enqueue history, the key will be processed later again.
	ctrl.queue.AddRateLimited(key)
}

func (ctrl *Controller) runWorker() {
	for ctrl.processNextItem() {
	}
}
func (ctrl *Controller) processNextItem() bool {
	key, quit := ctrl.queue.Get()
	if quit {
		return false
	}

	defer ctrl.queue.Done(key)
	err := ctrl.syncUserCluster()
	ctrl.handleErr(err, key)
	return true
}

// syncUserCluster will reconcile the user-cluster
func (ctrl *Controller) syncUserCluster() error {
	glog.V(6).Infof("Syncing user-cluster")

	// Get confimaps from lister, make a copy.
	cachedConfigMaps, err := ctrl.configMapLister.List(labels.Everything())
	if err != nil {
		return fmt.Errorf("failed to receive configMaps from lister: %v", err)
	}
	configMaps := make([]*corev1.ConfigMap, len(cachedConfigMaps))
	for i := range cachedConfigMaps {
		configMaps[i] = cachedConfigMaps[i].DeepCopy()
	}

	return nil
}
