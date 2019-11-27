package seed

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	"github.com/kubermatic/kubermatic/api/pkg/controller/util"
	predicateutil "github.com/kubermatic/kubermatic/api/pkg/controller/util/predicate"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// ControllerName is the name of this very controller.
	ControllerName = "kubermatic-seed-operator"

	// NameLabel is the label containing the application's name.
	NameLabel = "app.kubernetes.io/name"

	// VersionLabel is the label containing the application's version.
	VersionLabel = "app.kubernetes.io/version"
)

func Add(
	ctx context.Context,
	log *zap.SugaredLogger,
	namespace string,
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	seedsGetter provider.SeedsGetter,
	numWorkers int,
	workerName string,
) error {
	namespacePredicate := predicateutil.ByNamespace(namespace)

	reconciler := &Reconciler{
		ctx:            ctx,
		log:            log.Named(ControllerName),
		scheme:         masterManager.GetScheme(),
		namespace:      namespace,
		masterClient:   masterManager.GetClient(),
		masterRecorder: masterManager.GetEventRecorderFor(ControllerName),
		seedClients:    map[string]ctrlruntimeclient.Client{},
		seedRecorders:  map[string]record.EventRecorder{},
		seedsGetter:    seedsGetter,
		workerName:     workerName,
	}

	ctrlOpts := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, masterManager, ctrlOpts)
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	// watch for changes to KubermaticConfigurations in the master cluster
	obj := &operatorv1alpha1.KubermaticConfiguration{}
	configEventHandler := newEventHandler(func(_ handler.MapObject) []reconcile.Request {
		seeds, err := seedsGetter()
		if err != nil {
			log.Errorw("Failed to handle request", zap.Error(err))
			utilruntime.HandleError(err)
			return nil
		}

		requests := []reconcile.Request{}
		for _, seed := range seeds {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: seed.Name,
				},
			})
		}

		return requests
	})

	if err := c.Watch(&source.Kind{Type: obj}, configEventHandler, namespacePredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", obj, err)
	}

	// watch all resources we manage inside all configured seeds
	for key, manager := range seedManagers {
		reconciler.seedClients[key] = manager.GetClient()
		reconciler.seedRecorders[key] = manager.GetEventRecorderFor(ControllerName)

		if err := createSeedWatches(c, key, manager, namespacePredicate, common.ManagedByOperatorPredicate); err != nil {
			return fmt.Errorf("failed to setup watches for seed %s: %v", key, err)
		}
	}

	return nil
}

func createSeedWatches(controller controller.Controller, seedName string, seedManager manager.Manager, predicates ...predicate.Predicate) error {
	cache := seedManager.GetCache()
	eventHandler := util.EnqueueConst(seedName)

	typesToWatch := []runtime.Object{
		&appsv1.Deployment{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
		&kubermaticv1.Seed{},
	}

	for _, t := range typesToWatch {
		seedTypeWatch := &source.Kind{Type: t}
		if err := seedTypeWatch.InjectCache(cache); err != nil {
			return fmt.Errorf("failed to inject cache into watch for %T: %v", t, err)
		}
		if err := controller.Watch(seedTypeWatch, eventHandler, predicates...); err != nil {
			return fmt.Errorf("failed to watch %T: %v", t, err)
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
