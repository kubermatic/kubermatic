package usercluster

import (
	"context"
	"errors"
	"fmt"

	"github.com/heptiolabs/healthcheck"
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
func Add(mgr manager.Manager, openshift bool, registerReconciledCheck func(name string, check healthcheck.Check)) error {
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

	// A very simple but limited way to express the successful reconciling to the seed cluster
	registerReconciledCheck(fmt.Sprintf("%s-%s", controllerName, "reconciling_successfully"), func() error {
		if !reconcile.reconcilingSuccessfully {
			return errors.New("last reconcile failed or did not happen yet")
		}
		return nil
	})

	return nil
}

// reconcileUserCluster reconciles objects in the user cluster
type reconciler struct {
	client.Client
	openshift               bool
	cache                   cache.Cache
	reconcilingSuccessfully bool
}

// Reconcile makes changes in response to objects in the user cluster.
func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if err := r.reconcile(context.TODO()); err != nil {
		r.reconcilingSuccessfully = false
		return reconcile.Result{}, err
	}
	r.reconcilingSuccessfully = true

	return reconcile.Result{}, nil
}
