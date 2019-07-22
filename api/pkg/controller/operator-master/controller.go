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

	// InstanceLabel is the recommended label for distinguishing
	// multiple elements of the same name. The label is used to store
	// the seed cluster name.
	InstanceLabel = "app.kubernetes.io/instance"

	// ManagedByLabel is the label used to identify the resources
	// created by this controller.
	ManagedByLabel = "app.kubernetes.io/managed-by"
)

func Add(
	mgr manager.Manager,
	numWorkers int,
	clientConfig *clientcmdapi.Config,
	log *zap.SugaredLogger,
) error {
	reconciler := &Reconciler{
		Client:       mgr.GetClient(),
		recorder:     mgr.GetRecorder(ControllerName),
		clientConfig: clientConfig,
		log:          log,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	// use the object's namespace as the reconcile key
	eventHandler := &handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: a.Meta.GetNamespace(),
				},
			},
		}
	})}

	ownedByPred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return managedByController(e.Meta)
		},

		UpdateFunc: func(e event.UpdateEvent) bool {
			return managedByController(e.MetaOld) || managedByController(e.MetaNew)
		},

		DeleteFunc: func(e event.DeleteEvent) bool {
			return managedByController(e.Meta)
		},

		GenericFunc: func(e event.GenericEvent) bool {
			return managedByController(e.Meta)
		},
	}

	// watch all KubermaticConfigurations
	t := &operatorv1alpha1.KubermaticConfiguration{}
	if err := c.Watch(&source.Kind{Type: t}, eventHandler); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", t, err)
	}

	// watch all other kinds that are owned by the operator
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
		if err := c.Watch(&source.Kind{Type: t}, eventHandler, ownedByPred); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	return nil
}

func managedByController(meta metav1.Object) bool {
	labels := meta.GetLabels()
	return labels[ManagedByLabel] == ControllerName
}
