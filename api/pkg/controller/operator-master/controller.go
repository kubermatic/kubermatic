package operatormaster

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "kubermatic-master-operator"

	// NameLabel is the label containing the application's name.
	NameLabel = "app.kubernetes.io/name"

	// VersionLabel is the label containing the application's version.
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
	ctx context.Context,
	mgr manager.Manager,
	log *zap.SugaredLogger,
	namespace string,
	numWorkers int,
	workerName string,
) error {
	reconciler := &Reconciler{
		Client:     mgr.GetClient(),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		log:        log.Named(ControllerName),
		workerName: workerName,
		ctx:        ctx,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	namespacePredicate := predicateutil.ByNamespace(namespace)
	workerNamePredicate := workerlabel.Predicates(workerName)

	// put the config's identifier on the queue
	kubermaticConfigHandler := newEventHandler(func(a handler.MapObject) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: a.Meta.GetNamespace(),
					Name:      a.Meta.GetName(),
				},
			},
		}
	})

	obj := &operatorv1alpha1.KubermaticConfiguration{}
	if err := c.Watch(&source.Kind{Type: obj}, kubermaticConfigHandler, namespacePredicate, workerNamePredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", obj, err)
	}

	// for each child put the parent configuration onto the queue
	childEventHandler := newEventHandler(func(a handler.MapObject) []reconcile.Request {
		if a.Meta.GetLabels()[ManagedByLabel] != ControllerName {
			return nil
		}

		owner := a.Meta.GetAnnotations()[ConfigurationOwnerAnnotation]
		if owner == "" {
			return nil
		}

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
		if err := c.Watch(&source.Kind{Type: t}, childEventHandler, namespacePredicate); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	return nil
}

// newEventHandler takes a obj->request mapper function and wraps it into an
// handler.EnqueueRequestsFromMapFunc.
func newEventHandler(rf handler.ToRequestsFunc) *handler.EnqueueRequestsFromMapFunc {
	return &handler.EnqueueRequestsFromMapFunc{
		ToRequests: rf,
	}
}
