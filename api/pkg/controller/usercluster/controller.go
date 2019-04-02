package usercluster

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/golang/glog"
	"github.com/heptiolabs/healthcheck"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

const (
	controllerName = "user-cluster-controller"
)

// Add creates a new user cluster controller.
func Add(
	mgr manager.Manager,
	openshift bool,
	namespace string,
	caCert *x509.Certificate,
	clusterURL *url.URL,
	openvpnServerPort int,
	registerReconciledCheck func(name string, check healthcheck.Check)) error {
	reconcile := &reconciler{
		Client:            mgr.GetClient(),
		cache:             mgr.GetCache(),
		openshift:         openshift,
		rLock:             &sync.Mutex{},
		namespace:         namespace,
		caCert:            caCert,
		clusterURL:        clusterURL,
		openvpnServerPort: openvpnServerPort,
	}
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: reconcile})
	if err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &apiregistrationv1beta1.APIService{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &corev1.ServiceAccount{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &rbacv1.Role{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRole{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &rbacv1.ClusterRoleBinding{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &admissionregistrationv1beta1.MutatingWebhookConfiguration{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}

	if err = c.Watch(&source.Kind{Type: &apiextensionsv1beta1.CustomResourceDefinition{}}, &handler.EnqueueRequestForObject{}); err != nil {
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
	openshift         bool
	cache             cache.Cache
	namespace         string
	caCert            *x509.Certificate
	clusterURL        *url.URL
	openvpnServerPort int

	rLock                      *sync.Mutex
	reconciledSuccessfullyOnce bool
}

// Reconcile makes changes in response to objects in the user cluster.
func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if err := r.reconcile(context.TODO()); err != nil {
		glog.Errorf("Reconciling failed: %v", err)
		return reconcile.Result{}, err
	}

	r.rLock.Lock()
	defer r.rLock.Unlock()
	r.reconciledSuccessfullyOnce = true
	return reconcile.Result{}, nil
}
