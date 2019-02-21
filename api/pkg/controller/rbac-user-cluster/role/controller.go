package role

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "rbac-role-user-cluster-controller"
)

// Add creates a new Cluster Role generator controller that is responsible for creating Cluster Role for cluster resources
// and for groups: `owners`, `editors` and `viewers``
func Add(mgr manager.Manager) (string, error) {
	reconcile := &reconcileClusterRole{Client: mgr.GetClient(), ctx: context.TODO()}

	return controllerName, add(mgr, reconcile)
}

// add adds a new Controller to mgr with r as the Reconcile.reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to ClusterRoles
	err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}

// reconcileClusterRole reconciles Cluster Role objects
type reconcileClusterRole struct {
	ctx context.Context
	client.Client
}

// Reconcile makes changes in response to Cluster Role related changes
func (r *reconcileClusterRole) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	rdr := reconciler{client: r.Client, ctx: r.ctx}
	r.Get(r.ctx, request.NamespacedName, &rbacv1.ClusterRole{})
	err := rdr.Reconcile()

	return reconcile.Result{}, err
}
