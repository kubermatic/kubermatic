package master

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
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
		scheme:     mgr.GetScheme(),
		recorder:   mgr.GetEventRecorderFor(ControllerName),
		log:        log.Named(ControllerName),
		workerName: workerName,
		ctx:        ctx,
		versions:   common.NewDefaultVersions(),
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	namespacePredicate := predicateutil.ByNamespace(namespace)

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
	if err := c.Watch(&source.Kind{Type: cfg}, kubermaticConfigHandler, namespacePredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", cfg, err)
	}

	// for each child put the parent configuration onto the queue
	childEventHandler := newEventHandler(func(a handler.MapObject) []reconcile.Request {
		configs := &operatorv1alpha1.KubermaticConfigurationList{}
		options := &ctrlruntimeclient.ListOptions{Namespace: namespace}

		if err := mgr.GetClient().List(ctx, configs, options); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to list KubermaticConfigurations: %v", err))
			return nil
		}

		// when handling namespaces, it's okay to not find a KubermaticConfiguration
		// and simply skip reconciling
		if len(configs.Items) == 0 {
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
		&rbacv1.ClusterRoleBinding{},
		&policyv1beta1.PodDisruptionBudget{},
		&admissionregistrationv1beta1.ValidatingWebhookConfiguration{},
		&certmanagerv1alpha2.Certificate{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, childEventHandler, namespacePredicate, common.ManagedByOperatorPredicate); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %v", t, err)
		}
	}

	// namespaces are not managed by the operator and so can use neither namespacePredicate
	// nor ManagedByPredicate, but still need to get their labels reconciled
	ns := &corev1.Namespace{}
	if err := c.Watch(&source.Kind{Type: ns}, childEventHandler, predicateutil.ByName(namespace)); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", ns, err)
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
