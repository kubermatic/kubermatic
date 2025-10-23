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

	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type resourcesController struct {
	log              *zap.SugaredLogger
	metrics          *Metrics
	projectResources []projectResource
	client           ctrlruntimeclient.Client
	restMapper       meta.RESTMapper
	providerName     string
	objectType       ctrlruntimeclient.Object
}

// newResourcesController creates a new controller for managing RBAC for named resources that belong to project.
func newResourcesControllers(ctx context.Context, metrics *Metrics, mgr manager.Manager, log *zap.SugaredLogger, seedManagerMap map[string]manager.Manager, resources []projectResource) ([]*resourcesController, error) {
	log.Debug("considering master cluster provider for resources")

	for _, resource := range resources {
		rlog := log.With("kind", resource.object.GetObjectKind().GroupVersionKind().Kind)
		if resource.destination == destinationSeed {
			rlog.Debug("skipping adding a shared informer and indexer for master provider, as it is meant only for the seed cluster provider")
			continue
		}

		clonedObject := resource.object.DeepCopyObject().(ctrlruntimeclient.Object)

		mc := &resourcesController{
			log:              rlog,
			metrics:          metrics,
			projectResources: resources,
			client:           mgr.GetClient(),
			restMapper:       mgr.GetRESTMapper(),
			providerName:     "master",
			objectType:       clonedObject,
		}

		// Create a new controller
		_, err := builder.ControllerManagedBy(mgr).
			Named("rbac_generator_resources").
			For(clonedObject, builder.WithPredicates(predicateutil.Factory(resource.predicate))).
			Build(mc)
		if err != nil {
			return nil, err
		}
	}

	for seedName, seedManager := range seedManagerMap {
		seedLog := log.With("seed", seedName)
		seedLog.Debug("building controllers for seed", seedName)

		for _, resource := range resources {
			rlog := seedLog.With("kind", resource.object.GetObjectKind().GroupVersionKind().Kind)

			if len(resource.destination) == 0 {
				rlog.Debugf("skipping adding a shared informer and indexer, as it is meant only for the master cluster provider")
				continue
			}

			clonedObject := resource.object.DeepCopyObject().(ctrlruntimeclient.Object)

			c := &resourcesController{
				log:              rlog,
				metrics:          metrics,
				projectResources: resources,
				client:           seedManager.GetClient(),
				restMapper:       seedManager.GetRESTMapper(),
				providerName:     seedName,
				objectType:       clonedObject,
			}

			// Create a new controller
			_, err := builder.ControllerManagedBy(seedManager).
				Named(fmt.Sprintf("rbac_generator_resources_%s", seedName)).
				For(clonedObject, builder.WithPredicates(predicateutil.Factory(resource.predicate))).
				Build(c)
			if err != nil {
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

	if obj.GetObjectKind().GroupVersionKind().Empty() {
		if gvk, err := apiutil.GVKForObject(obj, c.client.Scheme()); err == nil {
			obj.GetObjectKind().SetGroupVersionKind(gvk)
		}
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
