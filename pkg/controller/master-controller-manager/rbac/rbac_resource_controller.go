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
	"context"
	"fmt"
	"time"

	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kcache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

type resourcesController struct {
	projectResourcesQueue workqueue.RateLimitingInterface
	ctx                   context.Context
	metrics               *Metrics
	projectResources      []projectResource
	client                client.Client
	restMapper            meta.RESTMapper
	providerName          string
	objectType            runtime.Object
}

type resourceToProcess struct {
	gvr        schema.GroupVersionResource
	kind       string
	metaObject metav1.Object
}

type queueItem struct {
	gvr      schema.GroupVersionResource
	kind     string
	name     string
	indexKey string
	cache    kcache.GenericLister
}

func (i *resourceToProcess) String() string {
	return i.metaObject.GetName()
}

// newResourcesController creates a new controller for managing RBAC for named resources that belong to project
func newResourcesControllers(ctx context.Context, metrics *Metrics, mgr manager.Manager, seedManagerMap map[string]manager.Manager, resources []projectResource) ([]*resourcesController, error) {
	// allControllers := []*resourcesController{mc}

	klog.V(4).Infof("considering master cluster provider for resources")
	for _, resource := range resources {
		clonedObject := resource.object.DeepCopyObject()

		mc := &resourcesController{
			projectResourcesQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_resources"),
			ctx:                   ctx,
			metrics:               metrics,
			projectResources:      resources,
			client:                mgr.GetClient(),
			restMapper:            mgr.GetRESTMapper(),
			providerName:          "master",
			objectType:            clonedObject,
		}

		// Create a new controller
		rcc, err := controller.New("rbac_generator_resources", mgr, controller.Options{Reconciler: mc})
		if err != nil {
			return nil, err
		}

		if resource.destination == destinationSeed {
			klog.V(4).Infof("skipping adding a shared informer and indexer for a project's resource %q for master provider, as it is meant only for the seed cluster provider", resource.object.GetObjectKind().GroupVersionKind().String())
			continue
		}

		if err = rcc.Watch(&source.Kind{Type: clonedObject}, &handler.EnqueueRequestForObject{}, predicateutil.Factory(resource.predicate)); err != nil {
			return nil, err
		}
	}

	for seedName, seedManager := range seedManagerMap {

		klog.V(4).Infof("considering %s provider for resources", seedName)
		for _, resource := range resources {
			clonedObject := resource.object.DeepCopyObject()

			c := &resourcesController{
				projectResourcesQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), fmt.Sprintf("rbac_generator_resources_%s", seedName)),
				ctx:                   ctx,
				metrics:               metrics,
				projectResources:      resources,
				client:                seedManager.GetClient(),
				restMapper:            seedManager.GetRESTMapper(),
				providerName:          seedName,
				objectType:            clonedObject,
			}

			// Create a new controller
			rc, err := controller.New(fmt.Sprintf("rbac_generator_resources_%s", seedName), seedManager, controller.Options{Reconciler: c})
			if err != nil {
				return nil, err
			}

			if len(resource.destination) == 0 {
				klog.V(4).Infof("skipping adding a shared informer and indexer for a project's resource %q for provider %q, as it is meant only for the master cluster provider", resource.object.GetObjectKind().GroupVersionKind().String(), seedName)
				continue
			}

			if err = rc.Watch(&source.Kind{Type: clonedObject}, &handler.EnqueueRequestForObject{}, predicateutil.Factory(resource.predicate)); err != nil {
				return nil, err
			}
		}

		// allControllers = append(allControllers, c)
	}

	return []*resourcesController{}, nil
}

func (c *resourcesController) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	err := c.syncProjectResource(req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

// run starts the controller's worker routines. This method is blocking and ends when stopCh gets closed
func (c *resourcesController) run(workerCount int, stopCh <-chan struct{}) {
	defer util.HandleCrash()

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
		gvr:        qItem.gvr,
		kind:       qItem.kind,
		metaObject: resMeta,
	}

	_ = processingItem

	// err = c.syncProjectResource(processingItem)
	c.handleErr(err, rawItem)
	return true
}

// func (c *resourcesController) enqueueProjectResource(obj interface{}, staticResource projectResource, lister kcache.GenericLister) {
// 	metaObj, err := meta.Accessor(obj)
// 	if err != nil {
// 		runtime.HandleError(fmt.Errorf("unable to get meta accessor for %#v, gvk %s, due to %v", obj, staticResource.object.GetObjectKind().GroupVersionKind().String(), err))
// 		return
// 	}
// 	if staticResource.shouldEnqueue != nil && !staticResource.shouldEnqueue(metaObj) {
// 		return
// 	}
// 	indexKey, err := kcache.MetaNamespaceKeyFunc(obj)
// 	if err != nil {
// 		runtime.HandleError(fmt.Errorf("unable to get the index key for %#v, gvr %s, due to %v", obj, staticResource.object.GetObjectKind().GroupVersionKind().String(), err))
// 		return
// 	}

// 	gvk := staticResource.object.GetObjectKind().GroupVersionKind()
// 	rmapping, err := c.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
// 	if err != nil {
// 		panic(err)
// 	}

// 	item := queueItem{
// 		gvr:      rmapping.Resource,
// 		kind:     staticResource.object.GetObjectKind().GroupVersionKind().Kind,
// 		name:     metaObj.GetName(),
// 		indexKey: indexKey,
// 		cache:    lister,
// 	}

// 	c.projectResourcesQueue.Add(item)
// }

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
