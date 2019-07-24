package operatormaster

import (
	"fmt"

	"go.uber.org/zap"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "kubermatic-master-operator"

	// NameLabel is the recommended name for an identifying label.
	NameLabel = "app.kubernetes.io/name"

	// VersionLabel is the recommended name for a version label.
	VersionLabel = "app.kubernetes.io/version"

	// ManagedByLabel is the label used to identify the resources
	// created by this controller.
	ManagedByLabel = "app.kubernetes.io/managed-by"

	// ConfigurationOwnerAnnotation is the annotation containing a resource's
	// owning configuration name and namespace.
	ConfigurationOwnerAnnotation = "operator.kubermatic.io/configuration"

	// WorkerNameLabel is the label containing the worker-name,
	// restricting the operator that is willing to work on a given
	// resource.
	WorkerNameLabel = "operator.kubermatic.io/worker"
)

func Add(
	mgr manager.Manager,
	numWorkers int,
	clientConfig *clientcmdapi.Config,
	log *zap.SugaredLogger,
	workerName string,
) error {
	reconciler := &Reconciler{
		Client:       mgr.GetClient(),
		recorder:     mgr.GetRecorder(ControllerName),
		clientConfig: clientConfig,
		log:          log.Named(ControllerName),
		workerName:   workerName,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	// reconcile for every KubermaticConfiguration labelled with the current worker name
	predicate := newPredicateFuncs(func(meta metav1.Object) bool {
		return meta.GetLabels()[WorkerNameLabel] == workerName
	})

	// put the config's identifier on the queue
	eventHandler := newEventHandler(func(a handler.MapObject) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: a.Meta.GetNamespace(),
					Name:      a.Meta.GetName(),
				},
			},
		}
	})

	t := &operatorv1alpha1.KubermaticConfiguration{}
	if err := c.Watch(&source.Kind{Type: t}, eventHandler, predicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", t, err)
	}

	// configure watches for all possibly controlled objects (filtering to only include those for the
	// current worker name is done by the reconciler later)
	predicate = newPredicateFuncs(func(meta metav1.Object) bool {
		return meta.GetLabels()[ManagedByLabel] == ControllerName
	})

	// put the owner onto the queue
	eventHandler = newEventHandler(func(a handler.MapObject) []reconcile.Request {
		owner := a.Meta.GetAnnotations()[ConfigurationOwnerAnnotation]

		ns, n, err := cache.SplitMetaNamespaceKey(owner)
		if err != nil {
			return nil
		}

		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: ns,
					Name:      n,
				},
			},
		}
	})

	typesToWatch := []runtime.Object{
		&appsv1.Deployment{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
		&extensionsv1beta1.Ingress{},
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, eventHandler, predicate); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	return nil
}

// newEventHandler takes a obj->request mapper function and wraps it into an
// handler.EnqueueRequestsFromMapFunc.
func newEventHandler(rf handler.ToRequestsFunc) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(rf),
	}
}

// newPredicateFuncs takes a predicate and returns a struct appliying the
// function for Create, Update, Delete and Generic events. For Update events,
// only the new version is considered.
func newPredicateFuncs(pred func(metav1.Object) bool) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return pred(e.Meta)
		},

		UpdateFunc: func(e event.UpdateEvent) bool {
			return pred(e.MetaNew)
		},

		DeleteFunc: func(e event.DeleteEvent) bool {
			return pred(e.Meta)
		},

		GenericFunc: func(e event.GenericEvent) bool {
			return pred(e.Meta)
		},
	}
}
