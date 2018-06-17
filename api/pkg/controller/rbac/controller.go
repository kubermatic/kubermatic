package rbac

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticsharedinformer "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticsharedinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	rbacinformer "k8s.io/client-go/informers/rbac/v1"
	"k8s.io/client-go/kubernetes"
	rbaclister "k8s.io/client-go/listers/rbac/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const (
	metricNamespace                   = "kubermatic"
	destinationSeed                   = "seed"
	dependantResyncTime time.Duration = 5 * time.Minute
)

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers prometheus.Gauge
}

// NewMetrics creates RBACGeneratorControllerMetrics
// with default values initialized, so metrics always show up.
func NewMetrics() *Metrics {
	subsystem := "rbac_generator_controller"
	cm := &Metrics{
		Workers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running RBACGenerator controller workers",
		}),
	}

	cm.Workers.Set(0)
	return cm
}

// Controller stores necessary components that are required to implement RBACGenerator
type Controller struct {
	projectQueue workqueue.RateLimitingInterface
	metrics      *Metrics
	workerName   string

	kubermaticMasterClient kubermaticclientset.Interface
	projectLister          kubermaticv1lister.ProjectLister
	projectSynced          cache.InformerSynced
	userLister             kubermaticv1lister.UserLister
	userSynced             cache.InformerSynced

	kubeMasterClient                kubernetes.Interface
	rbacClusterRoleLister           rbaclister.ClusterRoleLister
	rbacClusterRoleHasSynced        cache.InformerSynced
	rbacClusterRoleBindingLister    rbaclister.ClusterRoleBindingLister
	rbacClusterRoleBindingHasSynced cache.InformerSynced

	dependantInformers         []cache.Controller
	dependantInformesHasSynced []cache.InformerSynced
	dependantsQueue            workqueue.RateLimitingInterface

	seedClustersRESTClient []kubernetes.Interface
	projectResources       []projectResource
}

type projectResource struct {
	gvr         schema.GroupVersionResource
	kind        string
	destination string
}

// New creates a new RBACGenerator controller that is responsible for
// managing RBAC roles for project's resources
// The controller will also set proper ownership chain through OwnerReferences
// so that whenever a project is deleted dependants object will be garbage collected.
func New(
	metrics *Metrics,
	workerName string,
	kubermaticClient kubermaticclientset.Interface,
	kubermaticInformerFactory kubermaticsharedinformer.SharedInformerFactory,
	kubeClient kubernetes.Interface,
	rbacClusterRoleInformer rbacinformer.ClusterRoleInformer,
	rbacClusterRoleBindingInformer rbacinformer.ClusterRoleBindingInformer,
	seedClustersRESTClient []kubernetes.Interface) (*Controller, error) {
	c := &Controller{
		projectQueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "RBACGeneratorProject"),
		dependantsQueue:        workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "RBACGeneratorDependants"),
		metrics:                metrics,
		workerName:             workerName,
		kubermaticMasterClient: kubermaticClient,
		kubeMasterClient:       kubeClient,
		seedClustersRESTClient: seedClustersRESTClient,
	}

	prometheus.MustRegister(metrics.Workers)

	projectInformer := kubermaticInformerFactory.Kubermatic().V1().Projects()
	projectInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueProject(obj.(*kubermaticv1.Project))
		},
		UpdateFunc: func(old, cur interface{}) {
			c.enqueueProject(cur.(*kubermaticv1.Project))
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
			c.enqueueProject(project)
		},
	})
	c.projectLister = projectInformer.Lister()
	c.projectSynced = projectInformer.Informer().HasSynced

	userInformer := kubermaticInformerFactory.Kubermatic().V1().Users()
	c.userLister = userInformer.Lister()
	c.userSynced = userInformer.Informer().HasSynced

	c.rbacClusterRoleBindingLister = rbacClusterRoleBindingInformer.Lister()
	c.rbacClusterRoleBindingHasSynced = rbacClusterRoleBindingInformer.Informer().HasSynced

	c.rbacClusterRoleLister = rbacClusterRoleInformer.Lister()
	c.rbacClusterRoleHasSynced = rbacClusterRoleInformer.Informer().HasSynced

	// a list of dependent resources that we would like to watch/monitor
	c.projectResources = []projectResource{
		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.ClusterResourceName,
			},
			kind:        kubermaticv1.ClusterKindName,
			destination: destinationSeed,
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.SSHKeyResourceName,
			},
			kind: kubermaticv1.SSHKeyKind,
		},
	}

	for _, resource := range c.projectResources {
		// TODO: perhaps we should take into account destination and e.g don't watch master resources if the controller is being run on a seed cluster.
		informer, err := c.informerFor(kubermaticInformerFactory, resource.gvr, resource.kind)
		if err != nil {
			return nil, err
		}
		c.dependantInformers = append(c.dependantInformers, informer)
		c.dependantInformesHasSynced = append(c.dependantInformesHasSynced, informer.HasSynced)
	}

	return c, nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	glog.Infof("Starting RBACGenerator controller with %d workers", workerCount)
	defer glog.Info("Shutting down RBACGenerator controller")

	if !cache.WaitForCacheSync(stopCh, c.projectSynced, c.userSynced, c.rbacClusterRoleHasSynced, c.rbacClusterRoleBindingHasSynced) {
		runtime.HandleError(errors.New("Unable to sync caches for RBACGenerator controller"))
		return
	}
	if !cache.WaitForCacheSync(stopCh, c.dependantInformesHasSynced...) {
		runtime.HandleError(errors.New("Unable to sync caches for dependant resource of RBACGenerator controller"))
		return
	}

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runProjectWorker, time.Second, stopCh)
		go wait.Until(c.runDependantsWorker, time.Second, stopCh)
	}

	c.metrics.Workers.Set(float64(workerCount))
	<-stopCh
}

func (c *Controller) runProjectWorker() {
	for c.processProjectNextItem() {
	}
}

func (c *Controller) processProjectNextItem() bool {
	key, quit := c.projectQueue.Get()
	if quit {
		return false
	}
	defer c.projectQueue.Done(key)

	err := c.sync(key.(string))

	c.handleProjectErr(err, key)
	return true
}

// handleProjectErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleProjectErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.projectQueue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.projectQueue.NumRequeues(key) < 5 {
		glog.V(0).Infof("Error syncing %v: %v", key, err)

		// Re-enqueueProject the key rate limited. Based on the rate limiter on the
		// projectQueue and the re-enqueueProject history, the key will be processed later again.
		c.projectQueue.AddRateLimited(key)
		return
	}

	c.projectQueue.Forget(key)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.V(0).Infof("Dropping %q out of the projectQueue: %v", key, err)
}

func (c *Controller) enqueueProject(project *kubermaticv1.Project) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(project)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", project, err))
		return
	}

	c.projectQueue.Add(key)
}

func (c *Controller) runDependantsWorker() {
	for c.processDepentantsNextItem() {
	}
}

func (c *Controller) processDepentantsNextItem() bool {
	item, quit := c.dependantsQueue.Get()
	if quit {
		return false
	}
	defer c.projectQueue.Done(item)
	queueItem := item.(*dependantQueueItem)

	err := c.syncDependant(queueItem)

	c.handleDependantsErr(err, queueItem)
	return true
}

// handleDependantsErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleDependantsErr(err error, item *dependantQueueItem) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.dependantsQueue.Forget(item)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.dependantsQueue.NumRequeues(item) < 5 {
		glog.V(0).Infof("Error syncing gvr %s, kind %s, err %v", item.gvr.String(), item.kind, err)

		// Re-enqueueProject the key rate limited. Based on the rate limiter on the
		// projectQueue and the re-enqueueProject history, the key will be processed later again.
		c.dependantsQueue.AddRateLimited(item)
		return
	}

	c.dependantsQueue.Forget(item)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.V(0).Infof("Dropping %q out of the dependantsQueue: %v", item, err)
}

func (c *Controller) enqueueDependant(obj interface{}, gvr schema.GroupVersionResource, kind string) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("unable to get meta accessor for %#v, gvr %s", obj, gvr.String()))
	}
	item := &dependantQueueItem{
		gvr:        gvr,
		kind:       kind,
		metaObject: metaObj,
	}
	c.dependantsQueue.Add(item)
}

type dependantQueueItem struct {
	gvr        schema.GroupVersionResource
	kind       string
	metaObject metav1.Object
}

func (c *Controller) informerFor(sharedInformers kubermaticsharedinformers.SharedInformerFactory, gvr schema.GroupVersionResource, kind string) (cache.Controller, error) {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueDependant(obj, gvr, kind)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.enqueueDependant(newObj, gvr, kind)
		},
		DeleteFunc: func(obj interface{}) {
			if deletedFinalStateUnknown, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = deletedFinalStateUnknown.Obj
			}
			c.enqueueDependant(obj, gvr, kind)
		},
	}
	shared, err := sharedInformers.ForResource(gvr)
	if err == nil {
		glog.V(4).Infof("using a shared informer for dependant/resource %q", gvr.String())
		shared.Informer().AddEventHandlerWithResyncPeriod(handlers, dependantResyncTime)
		return shared.Informer().GetController(), nil
	}
	return nil, fmt.Errorf("uanble to create shared informer fo the given dependant/resourece %v", gvr.String())
}
