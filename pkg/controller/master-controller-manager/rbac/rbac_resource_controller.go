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

	"go.uber.org/zap"

	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/util/workqueue"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

type resourcesController struct {
	projectResourcesQueue workqueue.RateLimitingInterface
	log                   *zap.SugaredLogger
	metrics               *Metrics
	projectResources      []projectResource
	client                ctrlruntimeclient.Client
	restMapper            meta.RESTMapper
	providerName          string
	objectType            ctrlruntimeclient.Object
}

// newResourcesController creates a new controller for managing RBAC for named resources that belong to project.
func newResourcesControllers(ctx context.Context, metrics *Metrics, mgr manager.Manager, log *zap.SugaredLogger, seedManagerMap map[string]manager.Manager, resources []projectResource) ([]*resourcesController, error) {
	log.Debug("considering master cluster provider for resources")

	for _, resource := range resources {
		clonedObject := resource.object.DeepCopyObject()

		mc := &resourcesController{
			projectResourcesQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_resources"),
			log:                   log.With("kind", clonedObject.GetObjectKind().GroupVersionKind().Kind),
			metrics:               metrics,
			projectResources:      resources,
			client:                mgr.GetClient(),
			restMapper:            mgr.GetRESTMapper(),
			providerName:          "master",
			objectType:            clonedObject.(ctrlruntimeclient.Object),
		}

		// Create a new controller
		rcc, err := controller.New("rbac_generator_resources", mgr, controller.Options{Reconciler: mc})
		if err != nil {
			return nil, err
		}

		if resource.destination == destinationSeed {
			mc.log.Debug("skipping adding a shared informer and indexer for master provider, as it is meant only for the seed cluster provider")
			continue
		}

		if err = rcc.Watch(&source.Kind{Type: clonedObject.(ctrlruntimeclient.Object)}, &handler.EnqueueRequestForObject{}, predicateutil.Factory(resource.predicate)); err != nil {
			return nil, err
		}
	}

	for seedName, seedManager := range seedManagerMap {
		seedLog := log.With("seed", seedName)
		seedLog.Debug("building controllers for seed", seedName)

		for _, resource := range resources {
			clonedObject := resource.object.DeepCopyObject()

			c := &resourcesController{
				projectResourcesQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), fmt.Sprintf("rbac_generator_resources_%s", seedName)),
				log:                   seedLog.With("kind", clonedObject.GetObjectKind().GroupVersionKind().Kind),
				metrics:               metrics,
				projectResources:      resources,
				client:                seedManager.GetClient(),
				restMapper:            seedManager.GetRESTMapper(),
				providerName:          seedName,
				objectType:            clonedObject.(ctrlruntimeclient.Object),
			}

			// Create a new controller
			rc, err := controller.New(fmt.Sprintf("rbac_generator_resources_%s", seedName), seedManager, controller.Options{Reconciler: c})
			if err != nil {
				return nil, err
			}

			if len(resource.destination) == 0 {
				c.log.Debugf("skipping adding a shared informer and indexer, as it is meant only for the master cluster provider")
				continue
			}

			if err = rc.Watch(&source.Kind{Type: clonedObject.(ctrlruntimeclient.Object)}, &handler.EnqueueRequestForObject{}, predicateutil.Factory(resource.predicate)); err != nil {
				return nil, err
			}
		}
	}

	return []*resourcesController{}, nil
}

func (c *resourcesController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	obj := c.objectType.DeepCopyObject().(ctrlruntimeclient.Object)
	if err := c.client.Get(ctx, req.NamespacedName, obj); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, err
	}

	if obj.GetDeletionTimestamp() != nil {
		return reconcile.Result{}, nil
	}

	err := c.reconcile(ctx, obj)
	if err != nil {
		kind := obj.GetObjectKind().GroupVersionKind().Kind
		key := ctrlruntimeclient.ObjectKeyFromObject(obj)

		return reconcile.Result{}, fmt.Errorf("failed to reconcile %s %s in %s cluster: %w", kind, key, c.providerName, err)
	}

	return reconcile.Result{}, nil
}

func (c *resourcesController) reconcile(ctx context.Context, obj ctrlruntimeclient.Object) error {
	err := c.syncClusterScopedProjectResource(ctx, obj)
	if err != nil {
		return fmt.Errorf("failed to reconcile cluster-scoped resources: %w", err)
	}

	err = c.syncNamespaceScopedProjectResource(ctx, obj)
	if err != nil {
		return fmt.Errorf("failed to reconcile namespaced resources: %w", err)
	}

	err = c.syncClusterResource(ctx, obj)
	if err != nil {
		return fmt.Errorf("failed to sync Cluster resource: %w", err)
	}

	return nil
}
