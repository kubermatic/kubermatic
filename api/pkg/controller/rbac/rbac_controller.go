package rbac

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"

	kubermaticsharedinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
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
	projectQueue         workqueue.RateLimitingInterface
	projectBindingsQueue workqueue.RateLimitingInterface
	metrics              *Metrics

	projectLister            kubermaticv1lister.ProjectLister
	userLister               kubermaticv1lister.UserLister
	userProjectBindingLister kubermaticv1lister.UserProjectBindingLister

	projectResourcesInformers []cache.Controller
	projectResourcesQueue     workqueue.RateLimitingInterface

	seedClusterProviders  []*ClusterProvider
	masterClusterProvider *ClusterProvider

	projectResources []projectResource
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
func New(metrics *Metrics, allClusterProviders []*ClusterProvider) (*Controller, error) {
	c := &Controller{
		projectQueue:          workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_project"),
		projectBindingsQueue:  workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_project_bindings"),
		projectResourcesQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_project_resources"),
		metrics:               metrics,
	}

	for _, clusterProvider := range allClusterProviders {
		if strings.HasPrefix(clusterProvider.providerName, MasterProviderPrefix) {
			c.masterClusterProvider = clusterProvider
			break
		}
	}
	if c.masterClusterProvider == nil {
		return nil, errors.New("cannot create controller because master cluster provider has not been found")
	}

	// sets up an informer for project resources
	projectInformer := c.masterClusterProvider.kubermaticInformerFactory.Kubermatic().V1().Projects()
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
	userInformer := c.masterClusterProvider.kubermaticInformerFactory.Kubermatic().V1().Users()
	c.userLister = userInformer.Lister()
	c.userProjectBindingLister = c.masterClusterProvider.kubermaticInformerFactory.Kubermatic().V1().UserProjectBindings().Lister()

	// sets up an informer for project binding resources
	projectBindingsInformer := c.masterClusterProvider.kubermaticInformerFactory.Kubermatic().V1().UserProjectBindings()
	projectBindingsInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueProjectBinding(obj.(*kubermaticv1.UserProjectBinding))
		},
		UpdateFunc: func(old, cur interface{}) {
			c.enqueueProjectBinding(cur.(*kubermaticv1.UserProjectBinding))
		},
		DeleteFunc: func(obj interface{}) {
			projectBinding, ok := obj.(*kubermaticv1.UserProjectBinding)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					runtime.HandleError(fmt.Errorf("couldn't get object from tombstone %#v", obj))
					return
				}
				projectBinding, ok = tombstone.Obj.(*kubermaticv1.UserProjectBinding)
				if !ok {
					runtime.HandleError(fmt.Errorf("tombstone contained object that is not a UserProjectBinding %#v", obj))
					return
				}
			}
			c.enqueueProjectBinding(projectBinding)
		},
	})

	// sets up informers for project resources
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

	for _, clusterProvider := range allClusterProviders {
		glog.V(6).Infof("considering %s provider for resources", clusterProvider.providerName)
		for _, resource := range c.projectResources {
			if len(resource.destination) == 0 && !strings.HasPrefix(clusterProvider.providerName, MasterProviderPrefix) {
				glog.V(6).Infof("skipping adding a shared informer and indexer for a project's resource %q for provider %q, as it is meant only for the master cluster provider", resource.gvr.String(), clusterProvider.providerName)
				continue
			}
			if resource.destination == destinationSeed && !strings.HasPrefix(clusterProvider.providerName, SeedProviderPrefix) {
				glog.V(6).Infof("skipping adding a shared informer and indexer for a project's resource %q for provider %q, as it is meant only for the seed cluster provider", resource.gvr.String(), clusterProvider.providerName)
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

	for _, clusterProvider := range allClusterProviders {
		if strings.HasPrefix(clusterProvider.providerName, SeedProviderPrefix) {
			c.seedClusterProviders = append(c.seedClusterProviders, clusterProvider)
		}
	}

	return c, nil
}

// Run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *Controller) Run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runProjectWorker, time.Second, stopCh)
		go wait.Until(c.runProjectBindingsWorker, time.Second, stopCh)
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

	c.handleErr(err, key, c.projectQueue)
	return true
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *Controller) handleErr(err error, key interface{}, queue workqueue.RateLimitingInterface) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		queue.Forget(key)
		return
	}

	glog.V(0).Infof("Error syncing %v: %v", key, err)

	// Re-enqueue an item, based on the rate limiter on the
	// queue and the re-enqueueProject history, the key will be processed later again.
	queue.AddRateLimited(key)
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

	c.handleErr(err, queueItem, c.projectResourcesQueue)
	return true
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

func (c *Controller) runProjectBindingsWorker() {
	for c.processProjectBindingNextItem() {
	}
}

func (c *Controller) processProjectBindingNextItem() bool {
	key, quit := c.projectBindingsQueue.Get()
	if quit {
		return false
	}
	defer c.projectBindingsQueue.Done(key)

	err := c.syncProjectBindings(key.(string))

	c.handleErr(err, key, c.projectQueue)
	return true
}

func (c *Controller) enqueueProjectBinding(projectBinding *kubermaticv1.UserProjectBinding) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(projectBinding)
	if err != nil {
		runtime.HandleError(fmt.Errorf("couldn't get key for object %#v: %v", projectBinding, err))
		return
	}

	c.projectBindingsQueue.Add(key)
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
