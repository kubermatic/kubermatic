/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package rbac

import (
	"fmt"
	"strings"
	"time"

	kubermaticsharedinformers "github.com/kubermatic/kubermatic/api/pkg/crd/client/informers/externalversions"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kcache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

const projectResourcesResyncTime = 5 * time.Minute

type resourcesController struct {
	projectResourcesQueue workqueue.RateLimitingInterface

	metrics          *Metrics
	projectResources []projectResource
}

type resourceToProcess struct {
	gvr             schema.GroupVersionResource
	kind            string
	metaObject      metav1.Object
	clusterProvider *ClusterProvider
}

type queueItem struct {
	gvr             schema.GroupVersionResource
	kind            string
	name            string
	indexKey        string
	clusterProvider *ClusterProvider
	cache           kcache.GenericLister
}

func (i *resourceToProcess) String() string {
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
		klog.V(4).Infof("considering %s provider for resources", clusterProvider.providerName)
		for _, resource := range c.projectResources {
			if len(resource.destination) == 0 && !strings.HasPrefix(clusterProvider.providerName, MasterProviderPrefix) {
				klog.V(4).Infof("skipping adding a shared informer and indexer for a project's resource %q for provider %q, as it is meant only for the master cluster provider", resource.gvr.String(), clusterProvider.providerName)
				continue
			}
			if resource.destination == destinationSeed && !strings.HasPrefix(clusterProvider.providerName, SeedProviderPrefix) {
				klog.V(4).Infof("skipping adding a shared informer and indexer for a project's resource %q for provider %q, as it is meant only for the seed cluster provider", resource.gvr.String(), clusterProvider.providerName)
				continue
			}
			if resource.gvr.Group == kubermaticv1.GroupName {
				indexer, err := c.registerInformerIndexerForKubermaticResource(clusterProvider.kubermaticInformerFactory, resource, clusterProvider)
				if err != nil {
					return nil, err
				}
				clusterProvider.AddIndexerFor(indexer, resource.gvr)
				continue
			}
			err := c.registerInformerForKubeResource(clusterProvider.kubeInformerProvider, resource, clusterProvider)
			if err != nil {
				return nil, err
			}
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

	klog.Info("RBAC generator for resources controller started")
	<-stopCh
}

func (c *resourcesController) runProjectResourcesWorker() {
	for c.processProjectResourcesNextItem() {
	}
}

func (c *resourcesController) processProjectResourcesNextItem() bool {
	rawItem, quit := c.projectResourcesQueue.Get()
	if quit {
		return false
	}
	defer c.projectResourcesQueue.Done(rawItem)
	qItem := rawItem.(queueItem)

	runObj, err := qItem.cache.Get(qItem.indexKey)
	if err != nil {
		klog.V(4).Infof("won't process the resource %q because it's no longer in the queue", qItem.name)
		return true
	}
	resMeta, err := meta.Accessor(runObj)
	if err != nil {
		return true
	}
	processingItem := &resourceToProcess{
		gvr:             qItem.gvr,
		kind:            qItem.kind,
		clusterProvider: qItem.clusterProvider,
		metaObject:      resMeta,
	}

	err = c.syncProjectResource(processingItem)
	c.handleErr(err, rawItem)
	return true
}

func (c *resourcesController) enqueueProjectResource(obj interface{}, staticResource projectResource, clusterProvider *ClusterProvider, lister kcache.GenericLister) {
	metaObj, err := meta.Accessor(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("unable to get meta accessor for %#v, gvr %s, due to %v", obj, staticResource.gvr.String(), err))
		return
	}
	if staticResource.shouldEnqueue != nil && !staticResource.shouldEnqueue(metaObj) {
		return
	}
	indexKey, err := kcache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(fmt.Errorf("unable to get the index key for %#v, gvr %s, due to %v", obj, staticResource.gvr.String(), err))
		return
	}

	item := queueItem{
		gvr:             staticResource.gvr,
		kind:            staticResource.kind,
		name:            metaObj.GetName(),
		indexKey:        indexKey,
		clusterProvider: clusterProvider,
		cache:           lister,
	}

	c.projectResourcesQueue.Add(item)
}

func (c *resourcesController) registerInformerIndexerForKubermaticResource(sharedInformers kubermaticsharedinformers.SharedInformerFactory, resource projectResource, clusterProvider *ClusterProvider) (kcache.Indexer, error) {
	var genLister kcache.GenericLister

	handlers := kcache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueProjectResource(obj, resource, clusterProvider, genLister)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.enqueueProjectResource(newObj, resource, clusterProvider, genLister)
		},
		DeleteFunc: func(obj interface{}) {
			if deletedFinalStateUnknown, ok := obj.(kcache.DeletedFinalStateUnknown); ok {
				obj = deletedFinalStateUnknown.Obj
			}
			c.enqueueProjectResource(obj, resource, clusterProvider, genLister)
		},
	}
	shared, err := sharedInformers.ForResource(resource.gvr)
	if err == nil {
		klog.V(4).Infof("using a shared informer and indexer for %q resource, provider %q", resource.gvr.String(), clusterProvider.providerName)
		shared.Informer().AddEventHandlerWithResyncPeriod(handlers, projectResourcesResyncTime)
		genLister = shared.Lister()
		return shared.Informer().GetIndexer(), nil
	}
	return nil, fmt.Errorf("uanble to create shared informer and indexer for %q resource, provider %q, err %v", resource.gvr.String(), clusterProvider.providerName, err)
}

func (c *resourcesController) registerInformerForKubeResource(kubeInformerProvider InformerProvider, resource projectResource, clusterProvider *ClusterProvider) error {
	var genLister kcache.GenericLister

	handlers := kcache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c.enqueueProjectResource(obj, resource, clusterProvider, genLister)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			c.enqueueProjectResource(newObj, resource, clusterProvider, genLister)
		},
		DeleteFunc: func(obj interface{}) {
			if deletedFinalStateUnknown, ok := obj.(kcache.DeletedFinalStateUnknown); ok {
				obj = deletedFinalStateUnknown.Obj
			}
			c.enqueueProjectResource(obj, resource, clusterProvider, genLister)
		},
	}
	shared, err := kubeInformerProvider.KubeInformerFactoryFor(resource.namespace).ForResource(resource.gvr)
	if err == nil {
		klog.V(4).Infof("using a shared informer for %q resource, provider %q in namespace %q", resource.gvr.String(), clusterProvider.providerName, resource.namespace)
		shared.Informer().AddEventHandlerWithResyncPeriod(handlers, projectResourcesResyncTime)
		genLister = shared.Lister()

		if len(resource.namespace) > 0 {
			klog.V(4).Infof("registering Roles and RoleBindings informers in %q namespace for provider %s for resource %q", resource.namespace, clusterProvider.providerName, resource.gvr.String())
			_ = kubeInformerProvider.KubeInformerFactoryFor(resource.namespace).Rbac().V1().Roles().Lister()
			_ = kubeInformerProvider.KubeInformerFactoryFor(resource.namespace).Rbac().V1().RoleBindings().Lister()
		}
		return nil
	}
	return fmt.Errorf("uanble to create shared informer and indexer for the given project's resource %v for provider %q, err %v", resource.gvr.String(), clusterProvider.providerName, err)
}

// handleErr checks if an error happened and makes sure we will retry later.
func (c *resourcesController) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the #AddRateLimited history of the key on every successful synchronization.
		// This ensures that future processing of updates for this key is not delayed because of
		// an outdated error history.
		c.projectResourcesQueue.Forget(key)
		return
	}

	klog.Errorf("Error syncing %v: %v", key, err)

	// Re-enqueue an item, based on the rate limiter on the
	// queue and the re-enqueueProject history, the key will be processed later again.
	c.projectResourcesQueue.AddRateLimited(key)
}
