package master

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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

	cfg := &operatorv1alpha1.KubermaticConfiguration{}
	if err := c.Watch(&source.Kind{Type: cfg}, kubermaticConfigHandler, namespacePredicate, workerNamePredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", cfg, err)
	}

	// for each child put the parent configuration onto the queue
	childEventHandler := newEventHandler(func(a handler.MapObject) []reconcile.Request {
		if a.Meta.GetLabels()[common.ManagedByLabel] != common.OperatorName {
			return nil
		}

		configs := &operatorv1alpha1.KubermaticConfigurationList{}
		options := &ctrlruntimeclient.ListOptions{Namespace: namespace}

		if err := mgr.GetClient().List(ctx, configs, options); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list KubermaticConfigurations: %v", err))
			return nil
		}

		if len(configs.Items) == 0 {
			log.Warnw("could not find KubermaticConfiguration this object belongs to", "object", a)
			return nil
		}

		if len(configs.Items) > 1 {
			log.Warnw("found multiple KubermaticConfigurations in this namespace, refusing to guess the owner", "namespace", namespace)
			return nil
		}

		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Namespace: configs.Items[0].Namespace,
				Name:      configs.Items[0].Name,
			},
		}}
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
