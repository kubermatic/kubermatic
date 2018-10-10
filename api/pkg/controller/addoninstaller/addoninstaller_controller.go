package addoninstaller

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1informers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers prometheus.Gauge
}

// NewMetrics creates a new Metrics
// with default values initialized, so metrics always show up.
func NewMetrics() *Metrics {
	cm := &Metrics{
		Workers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "kubermatic",
			Subsystem: "addon_installer_controller",
			Name:      "workers",
			Help:      "The number of addon installer controller workers",
		}),
	}
	cm.Workers.Set(0)
	return cm
}

// Controller stores necessary components that are required to install in-cluster Add-On's
type Controller struct {
	workerName       string
	queue            workqueue.RateLimitingInterface
	metrics          *Metrics
	defaultAddonList []string
	client           kubermaticclientset.Interface
	clusterLister    kubermaticv1lister.ClusterLister
	addonLister      kubermaticv1lister.AddonLister
}

// New creates a new Addon-Installer controller that is responsible for
// installing in-cluster addons
func New(
	workerName string,
	metrics *Metrics,
	defaultAddonList []string,
	client kubermaticclientset.Interface,
	addonInformer kubermaticv1informers.AddonInformer,
	clusterInformer kubermaticv1informers.ClusterInformer) (*Controller, error) {

	c := &Controller{
		workerName:       workerName,
		queue:            workqueue.NewNamedRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(1*time.Second, 5*time.Minute), "addon_installer_cluster"),
		metrics:          metrics,
		defaultAddonList: defaultAddonList,
		client:           client,
	}

	prometheus.MustRegister(metrics.Workers)

	clusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueCluster(obj.(*kubermaticv1.Cluster))
		},
		UpdateFunc: func(old, cur interface{}) {
			c.enqueueCluster(cur.(*kubermaticv1.Cluster))
		},
		DeleteFunc: func(obj interface{}) {
			cluster, ok := obj.(*kubermaticv1.Cluster)
			// Object might be a tombstone
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					utilruntime.HandleError(fmt.Errorf("couldn't get obj from tombstone %#v", obj))
					return
				}
				cluster, ok = tombstone.Obj.(*kubermaticv1.Cluster)
				if !ok {
					utilruntime.HandleError(fmt.Errorf("tombstone contained object that is not a Cluster %#v", obj))
					return
				}
			}

			c.enqueueCluster(cluster)
		},
	})

	c.clusterLister = clusterInformer.Lister()
	c.addonLister = addonInformer.Lister()

	return c, nil
}

// If an clusterInformer triggers queuing, put the cluster into the workqeue
func (c *Controller) enqueueCluster(cluster *kubermaticv1.Cluster) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(cluster)

	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", cluster, err))
		return
	}
	c.queue.Add(key)
}

// make API call to create an addon in the cluster
func (c *Controller) createDefaultAddon(addon string, cluster *kubermaticv1.Cluster) error {
	gv := kubermaticv1.SchemeGroupVersion
	glog.V(8).Infof("Create addon %s for the cluster %s\n", addon, cluster.Name)

	a := &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:            addon,
			Namespace:       cluster.Status.NamespaceName,
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))},
			Labels:          map[string]string{},
		},
		Spec: kubermaticv1.AddonSpec{
			Name: addon,
			Cluster: corev1.ObjectReference{
				Name:       cluster.Name,
				Namespace:  "",
				UID:        cluster.UID,
				APIVersion: cluster.APIVersion,
				Kind:       "Cluster",
			},
		},
	}

	if c.workerName != "" {
		a.Labels[kubermaticv1.WorkerNameLabelKey] = c.workerName
	}

	if _, err := c.client.KubermaticV1().Addons(cluster.Status.NamespaceName).Create(a); err != nil {
		return err
	}

	err := wait.Poll(10*time.Millisecond, 10*time.Second, func() (bool, error) {
		_, err := c.addonLister.Addons(cluster.Status.NamespaceName).Get(a.Name)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return fmt.Errorf("failed waiting for addon %s to exist in the lister", a.Name)
	}

	return nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()

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

	if err := c.sync(key.(string)); err != nil {
		glog.V(0).Infof("Error syncing %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the
		// queue and the re-enqueue history, the key will be processed later again.
		c.queue.AddRateLimited(key)
		return true
	}

	// Forget about the #AddRateLimited history of the key on every successful synchronization.
	// This ensures that future processing of updates for this key is not delayed because of
	// an outdated error history.
	c.queue.Forget(key)
	return true
}

// make sure that all default addons are installed on cluster creation
func (c *Controller) sync(key string) error {
	clusterFromCache, err := c.clusterLister.Get(key)
	if err != nil {
		if kerrors.IsNotFound(err) {
			glog.V(2).Infof("cluster '%s' in work queue no longer exists", key)
			return nil
		}
		return fmt.Errorf("failed to get cluster from lister: %v", err)
	}

	cluster := clusterFromCache.DeepCopy()

	// Reconciling

	// Wait until the Apiserver is running to ensure the namespace exists at least.
	// Just checking for cluster.status.namespaceName is not enough as it gets set before the namespace exists
	if !cluster.Status.Health.Apiserver {
		glog.V(8).Infof("skipping addon sync for cluster %s as the apiserver is not running yet", key)
		c.queue.AddAfter(key, 1*time.Second)
		return nil
	}

	for _, defaultAddon := range c.defaultAddonList {
		_, err := c.addonLister.Addons(cluster.Status.NamespaceName).Get(defaultAddon)
		if err != nil && kerrors.IsNotFound(err) {
			if err = c.createDefaultAddon(defaultAddon, cluster); err != nil {
				return fmt.Errorf("failed to create initial adddon %s: %v", defaultAddon, err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to get addon %s: %v", defaultAddon, err)
		}
	}

	return err
}
