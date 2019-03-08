package rbacusercluster

import (
	"context"

	"k8s.io/apimachinery/pkg/types"

	"github.com/golang/glog"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName     = "rbac-user-cluster-controller"
	ResourceOwnerName  = "system:kubermatic:owners"
	ResourceEditorName = "system:kubermatic:editors"
	ResourceViewerName = "system:kubermatic:viewers"
)

var mapFn = handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      ResourceOwnerName,
			Namespace: "",
		}},
		{NamespacedName: types.NamespacedName{
			Name:      ResourceEditorName,
			Namespace: "",
		}},
		{NamespacedName: types.NamespacedName{
			Name:      ResourceViewerName,
			Namespace: "",
		}},
	}
})

// Add creates a new RBAC generator controller that is responsible for creating Cluster Roles and Cluster Role Bindings
// for groups: `owners`, `editors` and `viewers``
func Add(mgr manager.Manager) error {
	reconcile := &reconcileRBAC{Client: mgr.GetClient(), ctx: context.TODO()}

	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: reconcile})
	if err != nil {
		return err
	}

	// Watch for changes to ClusterRoles
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
		return err
	}
	// Watch for changes to ClusterRoleBindings
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
		return err
	}
	return nil
}

// reconcileRBAC reconciles Cluster Role and Cluster Role Binding objects
type reconcileRBAC struct {
	ctx context.Context
	client.Client
}

// Reconcile makes changes in response to Cluster Role and Cluster Role Binding related changes
func (r *reconcileRBAC) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rdr := reconciler{client: r.Client, ctx: r.ctx}

	err := rdr.Reconcile(request.Name)
	if err != nil {
		glog.Errorf("RBAC reconciliation failed: %v", err)
	}

	return reconcile.Result{}, err
}
