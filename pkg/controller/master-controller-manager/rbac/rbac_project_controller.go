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

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	metricNamespace = "kubermatic"
	destinationSeed = "seed"
)

type projectController struct {
	metrics *Metrics

	log              *zap.SugaredLogger
	projectResources []projectResource
	client           ctrlruntimeclient.Client
	restMapper       meta.RESTMapper
	seedClientMap    map[string]ctrlruntimeclient.Client
}

// newProjectRBACController creates a new controller that is responsible for
// managing RBAC roles for project's

// The controller will also set proper ownership chain through OwnerReferences
// so that whenever a project is deleted dependent object will be garbage collected.
func newProjectRBACController(ctx context.Context, metrics *Metrics, mgr manager.Manager, seedManagerMap map[string]manager.Manager, log *zap.SugaredLogger, resources []projectResource, workerPredicate predicate.Predicate) error {
	seedClientMap := make(map[string]ctrlruntimeclient.Client)
	for k, v := range seedManagerMap {
		seedClientMap[k] = v.GetClient()
	}

	c := &projectController{
		log:              log,
		metrics:          metrics,
		projectResources: resources,
		client:           mgr.GetClient(),
		restMapper:       mgr.GetRESTMapper(),
		seedClientMap:    seedClientMap,
	}

	// Create a new controller
	_, err := builder.ControllerManagedBy(mgr).
		Named("rbac_generator_for_project").
		For(&kubermaticv1.Project{}, builder.WithPredicates(workerPredicate)).
		Build(c)

	return err
}

func (c *projectController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, c.sync(ctx, req.NamespacedName)
}
