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
		log:          log,
		workerName:   workerName,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	// configure watches for all related objects
	pred := makeOwnerPredicate(workerName)
	eventHandler := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(eventHandler)}
	typesToWatch := []runtime.Object{
		&appsv1.Deployment{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
		&extensionsv1beta1.Ingress{},
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
		&operatorv1alpha1.KubermaticConfiguration{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, eventHandler, pred); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	return nil
}

// eventHandler translates an incoming event into queue items, depending
// on the affected object's type. The controller always puts the name
// and namespace of the related KubermaticConfiguration on the queue.
func eventHandler(a handler.MapObject) []reconcile.Request {
	name := types.NamespacedName{}

	if _, ok := a.Object.(*operatorv1alpha1.KubermaticConfiguration); ok {
		// put the configuration itself on the queue
		name.Name = a.Meta.GetName()
		name.Namespace = a.Meta.GetNamespace()
	} else {
		// put the object's supposed owning configuration on the queue
		owner := a.Meta.GetAnnotations()[ConfigurationOwnerAnnotation]

		ns, n, err := cache.SplitMetaNamespaceKey(owner)
		if err != nil {
			return nil
		}

		name.Name = n
		name.Namespace = ns
	}

	return []reconcile.Request{
		{
			NamespacedName: name,
		},
	}
}

// predicateFunc is a function that decides for a given object and its metadata
// whether or not the controller should trigger a reconcile loop for an incoming
// event.
type predicateFunc func(runtime.Object, metav1.Object) bool

// makeOwnerPredicate builds a new predicate using the given worker name. The
// predicate checks whether the affected object has the proper labels.
func makeOwnerPredicate(workerName string) predicate.Predicate {
	return makePredicateFuncs(func(obj runtime.Object, meta metav1.Object) bool {
		labels := meta.GetLabels()

		// only reconcile configurations with our given worker name label
		if _, ok := obj.(*operatorv1alpha1.KubermaticConfiguration); ok {
			return labels[WorkerNameLabel] == workerName
		}

		// only act on objects managed by this controller
		return labels[ManagedByLabel] == ControllerName
	})
}

// makePredicateFuncs takes a predicate and returns a struct appliying the
// function for Create, Update, Delete and Generic events. For Update events,
// only the new version is considered.
func makePredicateFuncs(pred predicateFunc) predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return pred(e.Object, e.Meta)
		},

		UpdateFunc: func(e event.UpdateEvent) bool {
			return pred(e.ObjectNew, e.MetaNew)
		},

		DeleteFunc: func(e event.DeleteEvent) bool {
			return pred(e.Object, e.Meta)
		},

		GenericFunc: func(e event.GenericEvent) bool {
			return pred(e.Object, e.Meta)
		},
	}
}
