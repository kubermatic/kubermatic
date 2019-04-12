package usercluster

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"

	"github.com/golang/glog"
	"github.com/heptiolabs/healthcheck"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
func Add(
	mgr manager.Manager,
	openshift bool,
	namespace string,
	caCert *x509.Certificate,
	clusterURL *url.URL,
	openvpnServerPort int,
	registerReconciledCheck func(name string, check healthcheck.Check),
	openVPNCA *resources.ECDSAKeyPair) error {
	reconciler := &reconciler{
		Client:            mgr.GetClient(),
		cache:             mgr.GetCache(),
		openshift:         openshift,
		namespace:         namespace,
		caCert:            caCert,
		clusterURL:        clusterURL,
		openvpnServerPort: openvpnServerPort,
		openVPNCA:         openVPNCA,
	}
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}

	mapFn := handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {
		return []reconcile.Request{
			{NamespacedName: types.NamespacedName{
				// There is no "parent object" like e.G. a cluster that can be used to reconcile, we just have a random set of resources
				// we reconcile one after another. To ensure we always have only one reconcile running at a time, we
				// use a static string as identifier
				Name:      "identifier",
				Namespace: "",
			}}}
	})

	typesToWatch := []runtime.Object{
		&apiregistrationv1beta1.APIService{},
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&corev1.ConfigMap{},
		&rbacv1.Role{},
		&rbacv1.RoleBinding{},
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
		&admissionregistrationv1beta1.MutatingWebhookConfiguration{},
		&apiextensionsv1beta1.CustomResourceDefinition{},
	}
	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
			return fmt.Errorf("failed to create watch for %T: %v", t, err)
		}
	}

	// A very simple but limited way to express the first successful reconciling to the seed cluster
	registerReconciledCheck(fmt.Sprintf("%s-%s", controllerName, "reconciled_successfully_once"), func() error {
		if !reconciler.reconciledSuccessfullyOnce {
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
	openVPNCA         *resources.ECDSAKeyPair

	reconciledSuccessfullyOnce bool
}

// Reconcile makes changes in response to objects in the user cluster.
func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if err := r.reconcile(context.TODO()); err != nil {
		glog.Errorf("Reconciling failed: %v", err)
		return reconcile.Result{}, err
	}

	r.reconciledSuccessfullyOnce = true
	return reconcile.Result{}, nil
}
