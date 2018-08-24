package rbac

import (
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
	metricNamespace = "kubermatic"
	destinationSeed = "seed"
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

	kubermaticMasterClient   kubermaticclientset.Interface
	projectLister            kubermaticv1lister.ProjectLister
	userLister               kubermaticv1lister.UserLister
	userProjectBindingLister kubermaticv1lister.UserProjectBindingLister

	kubeMasterClient                   kubernetes.Interface
	rbacClusterRoleMasterLister        rbaclister.ClusterRoleLister
	rbacClusterRoleBindingMasterLister rbaclister.ClusterRoleBindingLister

	projectResourcesInformers []cache.Controller
	projectResourcesQueue     workqueue.RateLimitingInterface

	seedClusterProviders []*ClusterProvider
	projectResources     []projectResource
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
	kubermaticMasterClient kubermaticclientset.Interface,
	kubermaticMasterInformerFactory kubermaticsharedinformer.SharedInformerFactory,
	kubeMasterClient kubernetes.Interface,
	rbacClusterRoleMasterInformer rbacinformer.ClusterRoleInformer,
	rbacClusterRoleBindingMasterInformer rbacinformer.ClusterRoleBindingInformer,
	seedClusterProviders []*ClusterProvider) (*Controller, error) {
	c := &Controller{
		projectQueue:           workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_project"),
		projectResourcesQueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_project_resources"),
		metrics:                metrics,
		kubermaticMasterClient: kubermaticMasterClient,
		kubeMasterClient:       kubeMasterClient,
		seedClusterProviders:   seedClusterProviders,
	}

	projectInformer := kubermaticMasterInformerFactory.Kubermatic().V1().Projects()
	prometheus.MustRegister(metrics.Workers)

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

	userInformer := kubermaticMasterInformerFactory.Kubermatic().V1().Users()
	c.userLister = userInformer.Lister()

	c.userProjectBindingLister = kubermaticMasterInformerFactory.Kubermatic().V1().UserProjectBindings().Lister()

	c.rbacClusterRoleBindingMasterLister = rbacClusterRoleBindingMasterInformer.Lister()
	c.rbacClusterRoleMasterLister = rbacClusterRoleMasterInformer.Lister()

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

		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.UserProjectBindingResourceName,
			},
			kind: kubermaticv1.UserProjectBindingKind,
		},
	}

	for _, clusterProvider := range seedClusterProviders {
		for _, resource := range c.projectResources {
			if len(resource.destination) == 0 && clusterProvider.providerName != masterProviderName {
				glog.V(6).Infof("skipping adding a shared informer and indexer for a project's resource %q for provider %q, as it is meant only for the master cluster provider", resource.gvr.String(), clusterProvider.providerName)
				continue
			}
			informer, indexer, err := c.informerIndexerFor(clusterProvider.kubermaticInformerFactory, resource.gvr, resource.kind, clusterProvider)
			if err != nil {
				return nil, err
			}
			clusterProvider.AddIndexerFor(indexer, resource.gvr)
			c.projectResourcesInformers = append(c.projectResourcesInformers, informer)
		}
	}

	return c, nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	for _, seedClusterProvider := range c.seedClusterProviders {
		err := seedClusterProvider.WaitForCachesToSync(stopCh)
		if err != nil {
			runtime.HandleError(err)
			return
		}
	}

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runProjectWorker, time.Second, stopCh)
		go wait.Until(c.runProjectResourcesWorker, time.Second, stopCh)
	}

	c.metrics.Workers.Set(float64(workerCount))
	glog.Info("RBACGenerator controller started")
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

func (c *Controller) runProjectResourcesWorker() {
	for c.processProjectResourcesNextItem() {
	}
}

func (c *Controller) processProjectResourcesNextItem() bool {
	item, quit := c.projectResourcesQueue.Get()
	if quit {
		return false
	}
	defer c.projectQueue.Done(item)
	queueItem := item.(*projectResourceQueueItem)

	err := c.syncProjectResource(queueItem)

	c.handleProjectResourcesErr(err, queueItem)
	return true
}

// handleProjectResourcesErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleProjectResourcesErr(err error, item *projectResourceQueueItem) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.projectResourcesQueue.Forget(item)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.projectResourcesQueue.NumRequeues(item) < 5 {
		glog.V(0).Infof("Error syncing gvr %s, kind %s, err %v", item.gvr.String(), item.kind, err)

		// Re-enqueueProject the key rate limited. Based on the rate limiter on the
		// projectQueue and the re-enqueueProject history, the key will be processed later again.
		c.projectResourcesQueue.AddRateLimited(item)
		return
	}

	c.projectResourcesQueue.Forget(item)
	// Report to an external entity that, even after several retries, we could not successfully process this key
	runtime.HandleError(err)
	glog.V(0).Infof("Dropping %q out of the projectResourcesQueue: %v", item, err)
}

func (c *Controller) enqueueProjectResource(obj interface{}, gvr schema.GroupVersionResource, kind string, clusterProvider *ClusterProvider) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("unable to get meta accessor for %#v, gvr %s", obj, gvr.String()))
	}
	item := &projectResourceQueueItem{
		gvr:             gvr,
		kind:            kind,
		metaObject:      metaObj,
		clusterProvider: clusterProvider,
	}
	c.projectResourcesQueue.Add(item)
}

const projectResourcesResyncTime time.Duration = 5 * time.Minute

type projectResourceQueueItem struct {
	gvr             schema.GroupVersionResource
	kind            string
	metaObject      metav1.Object
	clusterProvider *ClusterProvider
}

func (c *Controller) informerIndexerFor(sharedInformers kubermaticsharedinformers.SharedInformerFactory, gvr schema.GroupVersionResource, kind string, clusterProvider *ClusterProvider) (cache.Controller, cache.Indexer, error) {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueProjectResource(obj, gvr, kind, clusterProvider)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.enqueueProjectResource(newObj, gvr, kind, clusterProvider)
		},
		DeleteFunc: func(obj interface{}) {
			if deletedFinalStateUnknown, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = deletedFinalStateUnknown.Obj
			}
			c.enqueueProjectResource(obj, gvr, kind, clusterProvider)
		},
	}
	shared, err := sharedInformers.ForResource(gvr)
	if err == nil {
		glog.V(4).Infof("using a shared informer and indexer for a project's resource %q for provider %q", gvr.String(), clusterProvider.providerName)
		shared.Informer().AddEventHandlerWithResyncPeriod(handlers, projectResourcesResyncTime)
		return shared.Informer().GetController(), shared.Informer().GetIndexer(), nil
	}
	return nil, nil, fmt.Errorf("uanble to create shared informer and indexer for the given project's resource %v for provider %q", gvr.String(), clusterProvider.providerName)
}
