//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package controller

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"

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
	ControllerName = "group-project-binding-controller"
)

// Add creates a new group-project-binding controller and sets up Watches.
func Add(
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	numWorkers int,
	setOwnerRef bool,
) error {
	reconciler := &Reconciler{
		Client:      mgr.GetClient(),
		recorder:    mgr.GetEventRecorderFor(ControllerName),
		log:         log.Named(ControllerName),
		setOwnerRef: setOwnerRef,
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
		enqueueGroupProjectBindingsForRole(mgr.GetClient()),
		predicateutil.ByLabelExists(kubermaticv1.AuthZRoleLabel),
	); err != nil {
		return fmt.Errorf("failed to create ClusterRole watcher: %w", err)
	}

	// watch Roles with the authz.k8c.io/role label as we might need to create new ClusterRoleBindings/RoleBindings
	if err := c.Watch(
		&source.Kind{Type: &rbacv1.Role{}},
		enqueueGroupProjectBindingsForRole(mgr.GetClient()),
		predicateutil.ByLabelExists(kubermaticv1.AuthZRoleLabel),
	); err != nil {
		return fmt.Errorf("failed to create Role watcher: %w", err)
	}

	return nil
}

// enqueueGroupProjectBindingsForRole returns a handler.EventHandler that enqueues all GroupProjectBindings
// related to an observed ClusterRole/Role. The relationship is built via the authz.k8c.io/role label, which has to
// match the GroupProjectBinding.Spec.Role. Only GroupProjectBindings with a matching KKP role need to be reconciled
// when a new ClusterRole/Role object for that KKP role is created by rbac-controller.
func enqueueGroupProjectBindingsForRole(client ctrlruntimeclient.Client) handler.EventHandler {
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
