package certificate

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "kubermatic-master-certificate-operator"

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
) error {
	// wait for the cert-manager CRDs to appear
	err := wait.PollImmediateUntil(10*time.Second, func() (bool, error) {
		certCRD := &apiextensionsv1beta1.CustomResourceDefinition{}
		key := types.NamespacedName{
			Name: "certificates.cert-manager.io",
		}

		// must use API reader instead of client because the caches have not been started yet
		if err := mgr.GetAPIReader().Get(ctx, key, certCRD); err != nil {
			if kerrors.IsNotFound(err) {
				log.Debug("v1alpha2.cert-manager.io/Certificate CRD does not yet exist")
				return false, nil
			}

			return false, fmt.Errorf("failed to retrieve CRD: %v", err)
		}

		return true, nil
	}, ctx.Done())

	// someone has closed the context form the outside
	if err == wait.ErrWaitTimeout {
		return nil
	}

	// something bad happened
	if err != nil {
		return fmt.Errorf("failed to wait for cert-manager CRDs to exist: %v", err)
	}

	log.Info("cert-manager CRDs detected, starting certificate controller")

	reconciler := &Reconciler{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetEventRecorderFor(ControllerName),
		log:      log.Named(ControllerName),
		ctx:      ctx,
	}

	ctrlOptions := controller.Options{Reconciler: reconciler, MaxConcurrentReconciles: numWorkers}
	c, err := controller.New(ControllerName, mgr, ctrlOptions)
	if err != nil {
		return err
	}

	namespacePredicate := predicateutil.ByNamespace(namespace)

	// put the config's identifier on the queue
	kubermaticConfigHandler := newEventHandler(func(a handler.MapObject) []reconcile.Request {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Namespace: a.Meta.GetNamespace(),
				Name:      a.Meta.GetName(),
			},
		}}
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

	cert := &certmanagerv1alpha2.Certificate{}
	if err := c.Watch(&source.Kind{Type: cert}, childEventHandler, namespacePredicate, common.ManagedByOperatorPredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", cert, err)
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
