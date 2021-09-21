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

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	versionChangedPredicate := predicate.ResourceVersionChangedPredicate{}

	reconciler := &Reconciler{
		log:            log.Named(ControllerName),
		scheme:         masterManager.GetScheme(),
		namespace:      namespace,
		masterClient:   masterManager.GetClient(),
		masterRecorder: masterManager.GetEventRecorderFor(ControllerName),
		seedClients:    map[string]ctrlruntimeclient.Client{},
		seedRecorders:  map[string]record.EventRecorder{},
		seedsGetter:    seedsGetter,
		workerName:     workerName,
		versions:       kubermatic.NewDefaultVersions(),
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
	configEventHandler := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
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

	config := &operatorv1alpha1.KubermaticConfiguration{}
	if err := c.Watch(&source.Kind{Type: config}, configEventHandler, namespacePredicate, predicate.ResourceVersionChangedPredicate{}); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", config, err)
	}

	// watch for changes to the global CA bundle ConfigMap and replicate it into each Seed
	configMapEventHandler := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
		// find the owning KubermaticConfiguration
		config, err := getKubermaticConfigurationForNamespace(ctx, reconciler.masterClient, namespace, reconciler.log)
		if err != nil {
			log.Errorw("Failed to retrieve config", zap.Error(err))
			utilruntime.HandleError(err)
			return nil
		}
		if config == nil {
			return nil
		}

		defaulted, err := defaults.DefaultConfiguration(config, zap.NewNop().Sugar())
		if err != nil {
			log.Errorw("Failed to default config", zap.Error(err))
			utilruntime.HandleError(err)
			return nil
		}

		// we only care for one specific ConfigMap, but its name is dynamic so we cannot have
		// a static watch setup for it
		if a.GetName() != defaulted.Spec.CABundle.Name {
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

	configMap := &corev1.ConfigMap{}
	if err := c.Watch(&source.Kind{Type: configMap}, configMapEventHandler, namespacePredicate, versionChangedPredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", configMap, err)
	}

	// watch for changes to Seed CRs inside the master cluster and reconcile the seed itself only
	seed := &kubermaticv1.Seed{}
	if err := c.Watch(&source.Kind{Type: seed}, &handler.EnqueueRequestForObject{}, namespacePredicate, versionChangedPredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %v", seed, err)
	}

	// watch all resources we manage inside all configured seeds
	for key, manager := range seedManagers {
		reconciler.seedClients[key] = manager.GetClient()
		reconciler.seedRecorders[key] = manager.GetEventRecorderFor(ControllerName)

		if err := createSeedWatches(c, key, manager, namespace); err != nil {
			return fmt.Errorf("failed to setup watches for seed %s: %v", key, err)
		}
	}

	return nil
}

func createSeedWatches(controller controller.Controller, seedName string, seedManager manager.Manager, namespace string) error {
	cache := seedManager.GetCache()
	eventHandler := handler.EnqueueRequestsFromMapFunc(func(o ctrlruntimeclient.Object) []reconcile.Request {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Name:      seedName,
				Namespace: namespace,
			},
		}}
	})

	watch := func(t ctrlruntimeclient.Object, preds ...predicate.Predicate) error {
		seedTypeWatch := &source.Kind{Type: t}

		if err := seedTypeWatch.InjectCache(cache); err != nil {
			return fmt.Errorf("failed to inject cache into watch for %T: %v", t, err)
		}

		if err := controller.Watch(seedTypeWatch, eventHandler, preds...); err != nil {
			return fmt.Errorf("failed to watch %T: %v", t, err)
		}

		return nil
	}

	namespacedTypesToWatch := []ctrlruntimeclient.Object{
		&appsv1.Deployment{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
		&policyv1beta1.PodDisruptionBudget{},
	}

	for _, t := range namespacedTypesToWatch {
		if err := watch(t, predicateutil.ByNamespace(namespace), common.ManagedByOperatorPredicate); err != nil {
			return err
		}
	}

	globalTypesToWatch := []ctrlruntimeclient.Object{
		&rbacv1.ClusterRoleBinding{},
		&admissionregistrationv1.ValidatingWebhookConfiguration{},
	}

	for _, t := range globalTypesToWatch {
		if err := watch(t, common.ManagedByOperatorPredicate); err != nil {
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

	// The VPA gets resources deployed into the kube-system namespace.
	namespacedVPATypes := []ctrlruntimeclient.Object{
		&appsv1.Deployment{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
	}

	for _, t := range namespacedVPATypes {
		if err := watch(t, predicateutil.ByNamespace(metav1.NamespaceSystem), common.ManagedByOperatorPredicate); err != nil {
			return err
		}
	}

	globalVPATypes := []ctrlruntimeclient.Object{
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}

	for _, t := range globalVPATypes {
		if err := watch(t, common.ManagedByOperatorPredicate); err != nil {
			return err
		}
	}

	return nil
}

func getKubermaticConfigurationForNamespace(ctx context.Context, client ctrlruntimeclient.Client, namespace string, log *zap.SugaredLogger) (*operatorv1alpha1.KubermaticConfiguration, error) {
	// find the owning KubermaticConfiguration
	configList := &operatorv1alpha1.KubermaticConfigurationList{}
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace: namespace,
	}

	if err := client.List(ctx, configList, listOpts); err != nil {
		return nil, fmt.Errorf("failed to find KubermaticConfigurations: %v", err)
	}

	if len(configList.Items) == 0 {
		log.Debug("ignoring request for namespace without KubermaticConfiguration")
		return nil, nil
	}

	if len(configList.Items) > 1 {
		log.Infow("there are multiple KubermaticConfiguration objects, cannot reconcile", "namespace", namespace)
		return nil, nil
	}

	return &configList.Items[0], nil
}
