package usercluster

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"

	"github.com/heptiolabs/healthcheck"
	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "user-cluster-controller"
)

// Add creates a new user cluster controller.
func Add(
	mgr manager.Manager,
	seedMgr manager.Manager,
	openshift bool,
	version string,
	namespace string,
	cloudProviderName string,
	clusterURL *url.URL,
	openvpnServerPort int,
	registerReconciledCheck func(name string, check healthcheck.Check),
	cloudCredentialSecretTemplate *corev1.Secret,
	log *zap.SugaredLogger) error {
	reconciler := &reconciler{
		Client:                        mgr.GetClient(),
		seedClient:                    seedMgr.GetClient(),
		cache:                         mgr.GetCache(),
		openshift:                     openshift,
		version:                       version,
		rLock:                         &sync.Mutex{},
		namespace:                     namespace,
		clusterURL:                    clusterURL,
		openvpnServerPort:             openvpnServerPort,
		cloudCredentialSecretTemplate: cloudCredentialSecretTemplate,
		log:                           log,
		platform:                      cloudProviderName,
	}
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: reconciler})
	if err != nil {
		return err
	}

	mapFn := handler.ToRequestsFunc(func(o handler.MapObject) []reconcile.Request {
		log.Debugw("Controller got triggered",
			"type", fmt.Sprintf("%T", o.Object),
			"name", o.Meta.GetName(),
			"namespace", o.Meta.GetNamespace())
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

	if openshift {
		infrastructureConfigKind := &unstructured.Unstructured{}
		infrastructureConfigKind.SetKind("Infrastructure")
		infrastructureConfigKind.SetAPIVersion("config.openshift.io/v1")
		clusterVersionConfigKind := &unstructured.Unstructured{}
		clusterVersionConfigKind.SetKind("ClusterVersion")
		clusterVersionConfigKind.SetAPIVersion("config.openshift.io/v1")
		typesToWatch = append(typesToWatch, infrastructureConfigKind, clusterVersionConfigKind)
	}

	// Avoid getting triggered by the leader lease AKA: If the annotation exists AND changed on
	// UPDATE, do not reconcile
	predicateIgnoreLeaderLeaseRenew := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			if e.MetaOld.GetAnnotations()[resourcelock.LeaderElectionRecordAnnotationKey] == "" {
				return true
			}
			if e.MetaOld.GetAnnotations()[resourcelock.LeaderElectionRecordAnnotationKey] ==
				e.MetaNew.GetAnnotations()[resourcelock.LeaderElectionRecordAnnotationKey] {
				return true
			}
			return false
		},
	}
	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}, predicateIgnoreLeaderLeaseRenew); err != nil {
			return fmt.Errorf("failed to create watch for %T: %v", t, err)
		}
	}

	seedTypesToWatch := []runtime.Object{
		&corev1.Secret{},
	}
	for _, t := range seedTypesToWatch {
		seedWatch := &source.Kind{Type: t}
		if err := seedWatch.InjectCache(seedMgr.GetCache()); err != nil {
			return fmt.Errorf("failed to inject cache in seed cluster watch for %T: %v", t, err)
		}
		if err := c.Watch(seedWatch, &handler.EnqueueRequestsFromMapFunc{ToRequests: mapFn}); err != nil {
			return fmt.Errorf("failed to watch %T in seed: %v", t, err)
		}
	}

	// A very simple but limited way to express the first successful reconciling to the seed cluster
	registerReconciledCheck(fmt.Sprintf("%s-%s", controllerName, "reconciled_successfully_once"), func() error {
		reconciler.rLock.Lock()
		defer reconciler.rLock.Unlock()
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
	seedClient                    client.Client
	openshift                     bool
	version                       string
	cache                         cache.Cache
	namespace                     string
	clusterURL                    *url.URL
	openvpnServerPort             int
	platform                      string
	cloudCredentialSecretTemplate *corev1.Secret

	rLock                      *sync.Mutex
	reconciledSuccessfullyOnce bool

	log *zap.SugaredLogger
}

// Reconcile makes changes in response to objects in the user cluster.
func (r *reconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	if err := r.reconcile(context.TODO()); err != nil {
		r.log.Errorw("Reconciling failed", zap.Error(err))
		return reconcile.Result{}, err
	}

	r.rLock.Lock()
	defer r.rLock.Unlock()
	r.reconciledSuccessfullyOnce = true
	return reconcile.Result{}, nil
}

func (r *reconciler) caCert(ctx context.Context) (*triple.KeyPair, error) {
	return resources.GetClusterRootCA(ctx, r.namespace, r.seedClient)
}

func (r *reconciler) openVPNCA(ctx context.Context) (*resources.ECDSAKeyPair, error) {
	return resources.GetOpenVPNCA(ctx, r.namespace, r.seedClient)
}

func (r *reconciler) userSSHKeys(ctx context.Context) (map[string][]byte, error) {
	secret := &corev1.Secret{}
	if err := r.seedClient.Get(
		ctx,
		types.NamespacedName{Namespace: r.namespace, Name: resources.UserSSHKeys},
		secret,
	); err != nil {
		if kerrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return secret.Data, nil
}
