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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	metricNamespace = "kubermatic"
	destinationSeed = "seed"
)

type projectController struct {
	projectQueue workqueue.RateLimitingInterface
	metrics      *Metrics

	projectResources []projectResource
	client           client.Client
	restMapper       meta.RESTMapper
	seedClientMap    map[string]client.Client
}

// newProjectRBACController creates a new controller that is responsible for
// managing RBAC roles for project's

// The controller will also set proper ownership chain through OwnerReferences
// so that whenever a project is deleted dependants object will be garbage collected.
func newProjectRBACController(ctx context.Context, metrics *Metrics, mgr manager.Manager, seedManagerMap map[string]manager.Manager, resources []projectResource, workerPredicate predicate.Predicate) error {
	seedClientMap := make(map[string]client.Client)
	for k, v := range seedManagerMap {
		seedClientMap[k] = v.GetClient()
	}

	c := &projectController{
		projectQueue:     workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "rbac_generator_for_project"),
		metrics:          metrics,
		projectResources: resources,
		client:           mgr.GetClient(),
		restMapper:       mgr.GetRESTMapper(),
		seedClientMap:    seedClientMap,
	}

	// Create a new controller
	cc, err := controller.New("rbac_generator_for_project", mgr, controller.Options{Reconciler: c})
	if err != nil {
		return err
	}

	// Watch for changes to UserProjectBinding
	err = cc.Watch(&source.Kind{Type: &kubermaticv1.Project{}}, &handler.EnqueueRequestForObject{}, workerPredicate)
	if err != nil {
		return err
	}

	return nil
}

func (c *projectController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	err := c.sync(ctx, req.NamespacedName)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
