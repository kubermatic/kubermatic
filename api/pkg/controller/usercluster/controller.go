package usercluster

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	rbacv1 "k8s.io/api/rbac/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

const (
	controllerName = "user-cluster-controller"
)

// Add creates a new user cluster controller.
func Add(mgr manager.Manager, openshift bool) error {
	reconcile := &reconciler{Client: mgr.GetClient(), cache: mgr.GetCache(), openshift: openshift}
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: reconcile})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &apiregistrationv1beta1.APIService{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &rbacv1.Role{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	return nil
}

// reconcileUserCluster reconciles objects in the user cluster
type reconciler struct {
	client.Client
	openshift bool
	cache     cache.Cache
}

// Reconcile makes changes in response to objects in the user cluster.
func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, r.reconcile(context.TODO())
}
