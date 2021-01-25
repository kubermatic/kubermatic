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

	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type resourcesController struct {
	projectResourcesQueue workqueue.RateLimitingInterface
	metrics               *Metrics
	projectResources      []projectResource
	client                client.Client
	restMapper            meta.RESTMapper
	providerName          string
	objectType            runtime.Object
}

// newResourcesController creates a new controller for managing RBAC for named resources that belong to project
func newResourcesControllers(ctx context.Context, metrics *Metrics, mgr manager.Manager, seedManagerMap map[string]manager.Manager, resources []projectResource) ([]*resourcesController, error) {
	// allControllers := []*resourcesController{mc}

	klog.V(4).Infof("considering master cluster provider for resources")
	for _, resource := range resources {
		clonedObject := resource.object.DeepCopyObject()

		mc := &resourcesController{
			projectResourcesQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_resources"),
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

		if err = rcc.Watch(&source.Kind{Type: clonedObject.(client.Object)}, &handler.EnqueueRequestForObject{}, predicateutil.Factory(resource.predicate)); err != nil {
			return nil, err
		}
	}

	for seedName, seedManager := range seedManagerMap {

		klog.V(4).Infof("considering %s provider for resources", seedName)
		for _, resource := range resources {
			clonedObject := resource.object.DeepCopyObject()

			c := &resourcesController{
				projectResourcesQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), fmt.Sprintf("rbac_generator_resources_%s", seedName)),
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

			if err = rc.Watch(&source.Kind{Type: clonedObject.(client.Object)}, &handler.EnqueueRequestForObject{}, predicateutil.Factory(resource.predicate)); err != nil {
				return nil, err
			}
		}

		// allControllers = append(allControllers, c)
	}

	return []*resourcesController{}, nil
}

func (c *resourcesController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	err := c.syncProjectResource(ctx, req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
