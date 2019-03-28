package usercluster

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/heptiolabs/healthcheck"

	rbacv1 "k8s.io/api/rbac/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "user-cluster-controller"
)

// Add creates a new user cluster controller.
func Add(mgr manager.Manager, openshift bool, registerReconciledCheck func(name string, check healthcheck.Check)) error {
	reconcile := &reconciler{Client: mgr.GetClient(), cache: mgr.GetCache(), openshift: openshift, rLock: &sync.Mutex{}}
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

	// A very simple but limited way to express the first successful reconciling to the seed cluster
	registerReconciledCheck(fmt.Sprintf("%s-%s", controllerName, "reconciled_successfully_once"), func() error {
		reconcile.rLock.Lock()
		defer reconcile.rLock.Unlock()
		if !reconcile.reconciledSuccessfullyOnce {
			return errors.New("no successful reconciliation so far")
		}
		return nil
	})

	return nil
}

// reconcileUserCluster reconciles objects in the user cluster
type reconciler struct {
	client.Client
	openshift bool
	cache     cache.Cache

	rLock                      *sync.Mutex
	reconciledSuccessfullyOnce bool
}

// Reconcile makes changes in response to objects in the user cluster.
func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if err := r.reconcile(context.TODO()); err != nil {
		return reconcile.Result{}, err
	}

	r.rLock.Lock()
	defer r.rLock.Unlock()
	r.reconciledSuccessfullyOnce = true
	return reconcile.Result{}, nil
}
