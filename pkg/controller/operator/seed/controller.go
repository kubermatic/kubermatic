/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package seed

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	// This controller is responsible for creating/updating/deleting all the required resources on the seed clusters.
	ControllerName = "kkp-seed-operator"
)

func Add(
	log *zap.SugaredLogger,
	namespace string,
	masterManager manager.Manager,
	seedManagers map[string]manager.Manager,
	configGetter provider.KubermaticConfigurationGetter,
	seedsGetter provider.SeedsGetter,
	numWorkers int,
	workerName string,
) error {
	namespacePredicate := predicateutil.ByNamespace(namespace)
	workerNamePredicate := workerlabel.Predicate(workerName)
	versionChangedPredicate := predicate.ResourceVersionChangedPredicate{}

	// As the seedlifecyclecontroller skips uninitialized seeds, we do
	// the same so that we do not reconcile clusters for which we technically
	// should not do anything yet.
	seedsGetter = initializedSeedsGetter(seedsGetter)

	reconciler := &Reconciler{
		log:                    log.Named(ControllerName),
		scheme:                 masterManager.GetScheme(),
		namespace:              namespace,
		masterClient:           masterManager.GetClient(),
		masterRecorder:         masterManager.GetEventRecorder(ControllerName),
		seedClients:            map[string]ctrlruntimeclient.Client{},
		seedRecorders:          map[string]events.EventRecorder{},
		initializedSeedsGetter: seedsGetter,
		configGetter:           configGetter,
		workerName:             workerName,
		versions:               kubermatic.GetVersions(),
	}

	bldr := builder.ControllerManagedBy(masterManager).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		})

	// watch for changes to KubermaticConfigurations in the master cluster and reconcile all seeds
	configEventHandler := handler.EnqueueRequestsFromMapFunc(func(_ context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
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
					Name:      seed.Name,
					Namespace: seed.Namespace,
				},
			})
		}

		return requests
	})

	bldr.Watches(&kubermaticv1.KubermaticConfiguration{}, configEventHandler, builder.WithPredicates(namespacePredicate, workerNamePredicate, predicate.ResourceVersionChangedPredicate{}))

	// watch for changes to the global CA bundle ConfigMap and replicate it into each Seed
	configMapEventHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		// find the owning KubermaticConfiguration
		config, err := configGetter(ctx)
		if err != nil {
			log.Errorw("Failed to retrieve config", zap.Error(err))
			utilruntime.HandleError(err)
			return nil
		}
		if config == nil {
			return nil
		}

		if config.Labels[kubermaticv1.WorkerNameLabelKey] != workerName {
			log.Debugf("KubermaticConfiguration does not have matching %s label", kubermaticv1.WorkerNameLabelKey)
			return nil
		}

		// we only care for one specific ConfigMap, but its name is dynamic so we cannot have
		// a static watch setup for it
		if a.GetName() != config.Spec.CABundle.Name {
			return nil
		}

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
					Name:      seed.Name,
					Namespace: seed.Namespace,
				},
			})
		}

		return requests
	})

	bldr.Watches(&corev1.ConfigMap{}, configMapEventHandler, builder.WithPredicates(namespacePredicate, workerNamePredicate))

	// watch for changes to Seed CRs inside the master cluster and reconcile the seed itself only
	bldr.For(&kubermaticv1.Seed{}, builder.WithPredicates(namespacePredicate, workerNamePredicate, versionChangedPredicate))

	// watch all resources we manage inside all configured seeds (note that the seedManagers
	// map does not necessarily contain a manager for every seed, as uninitialized seeds
	// are automatically skipped by the seedlifecyclecontroller).
	for key, manager := range seedManagers {
		reconciler.seedClients[key] = manager.GetClient()
		reconciler.seedRecorders[key] = manager.GetEventRecorder(ControllerName)

		if err := createSeedWatches(bldr, key, manager, namespace, workerName); err != nil {
			return fmt.Errorf("failed to setup watches for seed %s: %w", key, err)
		}
	}

	_, err := bldr.Build(reconciler)

	return err
}

func createSeedWatches(bldr *builder.Builder, seedName string, seedManager manager.Manager, namespace string, workerName string) error {
	cache := seedManager.GetCache()
	eventHandler := handler.EnqueueRequestsFromMapFunc(func(_ context.Context, o ctrlruntimeclient.Object) []reconcile.Request {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Name:      seedName,
				Namespace: namespace,
			},
		}}
	})

	watch := func(t ctrlruntimeclient.Object, preds ...predicate.Predicate) {
		bldr.WatchesRawSource(source.Kind(cache, t, eventHandler, preds...))
	}

	namespacedTypesToWatch := []ctrlruntimeclient.Object{
		&appsv1.Deployment{},
		&batchv1.CronJob{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
		&policyv1.PodDisruptionBudget{},
	}

	for _, t := range namespacedTypesToWatch {
		watch(t, predicateutil.ByNamespace(namespace), common.ManagedByOperatorPredicate)
	}

	globalTypesToWatch := []ctrlruntimeclient.Object{
		&rbacv1.ClusterRoleBinding{},
		&admissionregistrationv1.ValidatingWebhookConfiguration{},
	}

	for _, t := range globalTypesToWatch {
		watch(t, common.ManagedByOperatorPredicate)
	}

	// Seeds are not managed by the operator, but we still need to be notified when
	// they are marked for deletion inside seed clusters
	watch(&kubermaticv1.Seed{}, predicateutil.ByNamespace(namespace), workerlabel.Predicate(workerName))

	// namespaces are not managed by the operator and so can use neither namespacePredicate
	// nor ManagedByPredicate, but still need to get their labels reconciled
	watch(&corev1.Namespace{}, predicateutil.ByName(namespace))

	// CRDs are not owned by KKP, but still need to be updated accordingly
	watch(&apiextensionsv1.CustomResourceDefinition{})

	// The VPA gets resources deployed into the kube-system namespace.
	namespacedVPATypes := []ctrlruntimeclient.Object{
		&appsv1.Deployment{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
	}

	for _, t := range namespacedVPATypes {
		watch(t, predicateutil.ByNamespace(metav1.NamespaceSystem), common.ManagedByOperatorPredicate)
	}

	globalVPATypes := []ctrlruntimeclient.Object{
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}

	for _, t := range globalVPATypes {
		watch(t, common.ManagedByOperatorPredicate)
	}

	return nil
}

// initializedSeedsGetter returns a seedsgetter that only returns
// initialized seeds.
func initializedSeedsGetter(seedsGetter provider.SeedsGetter) provider.SeedsGetter {
	return func() (map[string]*kubermaticv1.Seed, error) {
		seeds, err := seedsGetter()
		if err != nil {
			return nil, err
		}

		for name, seed := range seeds {
			if !seed.Status.IsInitialized() {
				delete(seeds, name)
			}
		}

		return seeds, nil
	}
}
