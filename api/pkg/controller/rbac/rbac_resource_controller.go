package rbac

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"

	kubermaticsharedinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

const projectResourcesResyncTime time.Duration = 5 * time.Minute

type resourcesController struct {
	projectResourcesInformers []cache.Controller
	projectResourcesQueue     workqueue.RateLimitingInterface

	metrics          *Metrics
	projectResources []projectResource
}

type projectResourceQueueItem struct {
	gvr             schema.GroupVersionResource
	kind            string
	namespace       string
	metaObject      metav1.Object
	clusterProvider *ClusterProvider
}

func (i *projectResourceQueueItem) String() string {
	return i.metaObject.GetName()
}

// newResourcesController creates a new controller for managing RBAC for named resources that belong to project
func newResourcesController(metrics *Metrics, allClusterProviders []*ClusterProvider, resources []projectResource) (*resourcesController, error) {
	c := &resourcesController{
		projectResourcesQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_resources"),
		metrics:               metrics,
		projectResources:      resources,
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
			if resource.gvr.Group == kubermaticv1.GroupName {
				informer, indexer, err := c.informerIndexerForKubermaticResource(clusterProvider.kubermaticInformerFactory, resource, clusterProvider)
				if err != nil {
					return nil, err
				}
				clusterProvider.AddIndexerFor(indexer, resource.gvr)
				c.projectResourcesInformers = append(c.projectResourcesInformers, informer)
				continue
			}
			informer, err := c.informerForKubeResource(clusterProvider.kubeInformerProvider, resource, clusterProvider)
			if err != nil {
				return nil, err
			}
			c.projectResourcesInformers = append(c.projectResourcesInformers, informer)
		}
	}

	return c, nil
}

// run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *resourcesController) run(workerCount int, stopCh <-chan struct{}) {
	defer runtime.HandleCrash()

	for i := 0; i < workerCount; i++ {
		go wait.Until(c.runProjectResourcesWorker, time.Second, stopCh)
		c.metrics.Workers.Inc()
	}

	glog.Info("RBAC generator for resources controller started")
	<-stopCh
}

func (c *resourcesController) runProjectResourcesWorker() {
	for c.processProjectResourcesNextItem() {
	}
}

func (c *resourcesController) processProjectResourcesNextItem() bool {
	item, quit := c.projectResourcesQueue.Get()
	if quit {
		return false
	}
	defer c.projectResourcesQueue.Done(item)
	queueItem := item.(*projectResourceQueueItem)

	err := c.syncProjectResource(queueItem)

	handleErr(err, queueItem, c.projectResourcesQueue)
	return true
}

func (c *resourcesController) enqueueProjectResource(obj interface{}, staticResource projectResource, clusterProvider *ClusterProvider) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("unable to get meta accessor for %#v, gvr %s", obj, staticResource.gvr.String()))
	}
	if staticResource.shouldEnqueue != nil && !staticResource.shouldEnqueue(metaObj) {
		return
	}
	item := &projectResourceQueueItem{
		gvr:             staticResource.gvr,
		kind:            staticResource.kind,
		metaObject:      metaObj,
		namespace:       metaObj.GetNamespace(),
		clusterProvider: clusterProvider,
	}
	c.projectResourcesQueue.Add(item)
}

func (c *resourcesController) informerIndexerForKubermaticResource(sharedInformers kubermaticsharedinformers.SharedInformerFactory, resource projectResource, clusterProvider *ClusterProvider) (cache.Controller, cache.Indexer, error) {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueProjectResource(obj, resource, clusterProvider)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.enqueueProjectResource(newObj, resource, clusterProvider)
		},
		DeleteFunc: func(obj interface{}) {
			if deletedFinalStateUnknown, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = deletedFinalStateUnknown.Obj
			}
			c.enqueueProjectResource(obj, resource, clusterProvider)
		},
	}
	shared, err := sharedInformers.ForResource(resource.gvr)
	if err == nil {
		glog.V(4).Infof("using a shared informer and indexer for %q resource, provider %q", resource.gvr.String(), clusterProvider.providerName)
		shared.Informer().AddEventHandlerWithResyncPeriod(handlers, projectResourcesResyncTime)
		return shared.Informer().GetController(), shared.Informer().GetIndexer(), nil
	}
	return nil, nil, fmt.Errorf("uanble to create shared informer and indexer for %q resource, provider %q, err %v", resource.gvr.String(), clusterProvider.providerName, err)
}

func (c *resourcesController) informerForKubeResource(kubeInformerProvider InformerProvider, resource projectResource, clusterProvider *ClusterProvider) (cache.Controller, error) {
	handlers := cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueProjectResource(obj, resource, clusterProvider)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.enqueueProjectResource(newObj, resource, clusterProvider)
		},
		DeleteFunc: func(obj interface{}) {
			if deletedFinalStateUnknown, ok := obj.(cache.DeletedFinalStateUnknown); ok {
				obj = deletedFinalStateUnknown.Obj
			}
			c.enqueueProjectResource(obj, resource, clusterProvider)
		},
	}
	shared, err := kubeInformerProvider.KubeInformerFactoryFor(resource.namespace).ForResource(resource.gvr)
	if err == nil {
		glog.V(4).Infof("using a shared informer for %q resource, provider %q in namespace %q", resource.gvr.String(), clusterProvider.providerName, resource.namespace)
		shared.Informer().AddEventHandlerWithResyncPeriod(handlers, projectResourcesResyncTime)

		if len(resource.namespace) > 0 {
			glog.V(4).Infof("registering Roles and RoleBindings informers in %q namespace for provider %s for resource %q", resource.namespace, clusterProvider.providerName, resource.gvr.String())
			_ = kubeInformerProvider.KubeInformerFactoryFor(resource.namespace).Rbac().V1().Roles().Lister()
			_ = kubeInformerProvider.KubeInformerFactoryFor(resource.namespace).Rbac().V1().RoleBindings().Lister()
		}
		return shared.Informer().GetController(), nil
	}
	return nil, fmt.Errorf("uanble to create shared informer and indexer for the given project's resource %v for provider %q, err %v", resource.gvr.String(), clusterProvider.providerName, err)
}
