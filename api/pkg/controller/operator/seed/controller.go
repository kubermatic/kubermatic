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

	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
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
		versions:       common.NewDefaultVersions(),
	}

	ctrlOpts := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, masterManager, ctrlOpts)
	if err != nil {
		return fmt.Errorf("failed to construct controller: %v", err)
	}

	// watch for changes to KubermaticConfigurations in the master cluster and reconcile all seeds
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

	config := &operatorv1alpha1.KubermaticConfiguration{}
	if err := c.Watch(&source.Kind{Type: config}, configEventHandler, namespacePredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", config, err)
	}

	// watch for changes to Seed CRs inside the master cluster and reconcile the seed itself only
	seed := &kubermaticv1.Seed{}
	if err := c.Watch(&source.Kind{Type: seed}, &handler.EnqueueRequestForObject{}, namespacePredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", seed, err)
	}

	// watch all resources we manage inside all configured seeds
	for key, manager := range seedManagers {
		reconciler.seedClients[key] = manager.GetClient()
		reconciler.seedRecorders[key] = manager.GetEventRecorderFor(ControllerName)

		if err := createSeedWatches(c, key, manager, namespace, namespacePredicate, common.ManagedByOperatorPredicate); err != nil {
			return fmt.Errorf("failed to setup watches for seed %s: %v", key, err)
		}
	}

	return nil
}

func createSeedWatches(controller controller.Controller, seedName string, seedManager manager.Manager, namespace string, predicates ...predicate.Predicate) error {
	cache := seedManager.GetCache()
	eventHandler := util.EnqueueConst(seedName)

	watch := func(t runtime.Object, preds ...predicate.Predicate) error {
		seedTypeWatch := &source.Kind{Type: t}

		if err := seedTypeWatch.InjectCache(cache); err != nil {
			return fmt.Errorf("failed to inject cache into watch for %T: %v", t, err)
		}

		if err := controller.Watch(seedTypeWatch, eventHandler, preds...); err != nil {
			return fmt.Errorf("failed to watch %T: %v", t, err)
		}

		return nil
	}

	typesToWatch := []runtime.Object{
		&appsv1.Deployment{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
		&rbacv1.ClusterRoleBinding{},
		&policyv1beta1.PodDisruptionBudget{},
		&admissionregistrationv1beta1.ValidatingWebhookConfiguration{},
	}

	for _, t := range typesToWatch {
		if err := watch(t, predicates...); err != nil {
			return err
		}
	}

	// Seeds are not managed by the operator, but we still need to be notified when
	// they are marked for deletion inside seed clusters
	if err := watch(&kubermaticv1.Seed{}, predicateutil.ByNamespace(namespace)); err != nil {
		return err
	}

	// namespaces are not managed by the operator and so can use neither namespacePredicate
	// nor ManagedByPredicate, but still need to get their labels reconciled
	if err := watch(&corev1.Namespace{}, predicateutil.ByName(namespace)); err != nil {
		return err
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
