package usercluster

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

const (
	controllerName = "user-cluster-controller"
)

var mapFn = handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {
	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{
			Name:      "reconcile-user-cluster-resources",
			Namespace: "",
		}},
	}
})

// Add creates a new user cluster controller.
func Add(mgr manager.Manager) (string, error) {
	reconcile := &reconcileUserCluster{Client: mgr.GetClient(), ctx: context.TODO()}
	return controllerName, add(mgr, reconcile)
}

// add adds a new Controller to mgr with r as the reconcile.reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &apiregistrationv1beta1.APIService{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &rbacv1.Role{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
		return err
	}

	return nil
}

// reconcileUserCluster reconciles objects in the user cluster
type reconcileUserCluster struct {
	ctx context.Context
	client.Client
}

// Reconcile makes changes in response to objects in the user cluster.
func (r *reconcileUserCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// TODO: reconcile other resources in the user cluster too
	rUserCluster := reconciler{client: r.Client, ctx: r.ctx}
	err := rUserCluster.Reconcile()
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
