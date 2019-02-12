package usercluster

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	admissionv1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

const (
	controllerName   = "user-cluster-controller"
	requeueAfter1sec = time.Second
)

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

	if err = c.Watch(&source.Kind{Type: &rbacv1.Role{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &rbacv1.RoleBinding{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &corev1.ConfigMap{}}, &handler.EnqueueRequestForObject{}); err != nil {
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
	if err = c.Watch(&source.Kind{Type: &apiextensionv1beta1.CustomResourceDefinition{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &apiregistrationv1beta1.APIService{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{}); err != nil {
		return err
	}
	if err = c.Watch(&source.Kind{Type: &admissionv1alpha1.InitializerConfiguration{}}, &handler.EnqueueRequestForObject{}); err != nil {
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
	// TODO: figure out how to pass the complete cluster object
	// and the required data to the Reconcile method
	// rUserCluster := reconciler{client: r.Client, ctx: r.ctx}
	// err := rUserCluster.Reconcile(cluster, data)
	// if err != nil {
	// 		return reconcile.Result{
	//  there is no need to set Requeue to true at the same time as RequeueAfter
	// TODO: figure out how to requeue in exponential manner
	//			RequeueAfter: requeueAfter1sec,
	//		}, err
	// }

	return reconcile.Result{}, nil
}
