/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package grouprbac

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller is responsible for synchronizing `GroupProjectBindings` to Kubernetes RBAC.
	ControllerName = "group-rbac-controller"
)

// Add creates a new group-rbac controller and sets up Watches.
func Add(
	ctx context.Context,
	mgr manager.Manager,
	numWorkers int,
	log *zap.SugaredLogger,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	seedsGetter provider.SeedsGetter,
) error {
	reconciler := &Reconciler{
		Client:           mgr.GetClient(),
		recorder:         mgr.GetEventRecorderFor(ControllerName),
		log:              log.Named(ControllerName),
		seedClientGetter: provider.SeedClientGetterFactory(seedKubeconfigGetter),
		seedsGetter:      seedsGetter,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	// watch all GroupProjectBindings
	if err := c.Watch(&source.Kind{Type: &kubermaticv1.GroupProjectBinding{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return fmt.Errorf("failed to create GroupProjectBinding watcher: %w", err)
	}

	// watch ClusterRoles with the authz.k8c.io/role label as we might need to create new ClusterRoleBindings/RoleBindings
	if err := c.Watch(
		&source.Kind{Type: &rbacv1.ClusterRole{}},
		enqueueGroupProjectBindingsForClusterRole(mgr.GetClient()),
		predicateutil.ByLabelExists(kubermaticv1.AuthZRoleLabel),
	); err != nil {
		return fmt.Errorf("failed to create ClusterRole watcher: %w", err)
	}

	return nil
}

// enqueueGroupProjectBindingsForClusterRole returns a handler.EventHandler that enqueues all GroupProjectBindings
// related to an observed ClusterRole. The relationship is built via the authz.k8c.io/role label, which has to
// match the GroupProjectBinding.Spec.Role. Only GroupProjectBindings with a matching KKP role need to be reconciled
// when a new ClusterRole object for that KKP role is created by rbac-controller.
func enqueueGroupProjectBindingsForClusterRole(client ctrlruntimeclient.Client) handler.EventHandler {
	return handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		var (
			requests []reconcile.Request
		)

		bindingList := &kubermaticv1.GroupProjectBindingList{}
		listOpts := &ctrlruntimeclient.ListOptions{}

		if err := client.List(context.Background(), bindingList, listOpts); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list GroupProjectBindings: %w", err))
			return []reconcile.Request{}
		}

		for _, binding := range bindingList.Items {
			if val, ok := a.GetLabels()[kubermaticv1.AuthZRoleLabel]; ok && strings.HasPrefix(val, binding.Spec.Role) {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name: binding.Name,
					},
				})
			}
		}

		return requests
	})
}
