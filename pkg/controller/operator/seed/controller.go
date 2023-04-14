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

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/pkg/controller/operator/seed/resources"
	controllerutil "k8c.io/kubermatic/v3/pkg/controller/util"
	predicateutil "k8c.io/kubermatic/v3/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v3/pkg/provider"
	"k8c.io/kubermatic/v3/pkg/util/workerlabel"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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
	ctx context.Context,
	log *zap.SugaredLogger,
	namespace string,
	mgr manager.Manager,
	configGetter provider.KubermaticConfigurationGetter,
	numWorkers int,
	workerName string,
	versions kubermatic.Versions,
) error {
	namespacePredicate := predicateutil.ByNamespace(namespace)
	workerNamePredicate := workerlabel.Predicates(workerName)
	versionChangedPredicate := predicate.ResourceVersionChangedPredicate{}

	reconciler := &Reconciler{
		log:          log.Named(ControllerName),
		scheme:       mgr.GetScheme(),
		namespace:    namespace,
		seedClient:   mgr.GetClient(),
		seedRecorder: mgr.GetEventRecorderFor(ControllerName),
		configGetter: configGetter,
		workerName:   workerName,
		versions:     versions,
	}

	ctrlOpts := controller.Options{
		Reconciler:              reconciler,
		MaxConcurrentReconciles: numWorkers,
	}
	c, err := controller.New(ControllerName, mgr, ctrlOpts)
	if err != nil {
		return fmt.Errorf("failed to construct controller: %w", err)
	}

	eventHandler := controllerutil.EnqueueConst("")

	config := &kubermaticv1.KubermaticConfiguration{}
	if err := c.Watch(&source.Kind{Type: config}, eventHandler, namespacePredicate, workerNamePredicate, versionChangedPredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %w", config, err)
	}

	// watch for changes to the global CA bundle ConfigMap and update the volume revision labels on the dependent deployments
	configMapEventHandler := handler.EnqueueRequestsFromMapFunc(func(a ctrlruntimeclient.Object) []reconcile.Request {
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

		if config.Labels[workerlabel.LabelKey] != workerName {
			log.Debugf("KubermaticConfiguration does not have matching %s label", workerlabel.LabelKey)
			return nil
		}

		// we only care for one specific ConfigMap, but its name is dynamic so we cannot have
		// a static watch setup for it
		if a.GetName() != config.Spec.CABundle.Name {
			return nil
		}

		// no need to enqueue anything specific
		return []reconcile.Request{{}}
	})

	configMap := &corev1.ConfigMap{}
	if err := c.Watch(&source.Kind{Type: configMap}, configMapEventHandler, namespacePredicate, versionChangedPredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %w", configMap, err)
	}

	// watch the namespaced resources we create, so we can undo unwnted modifications
	typesToWatch := []ctrlruntimeclient.Object{
		&appsv1.Deployment{},
		&batchv1.CronJob{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.ServiceAccount{},
		&corev1.Service{},
		&networkingv1.Ingress{},
		&policyv1.PodDisruptionBudget{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, eventHandler, namespacePredicate, resources.ManagedByOperatorPredicate); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %w", t, err)
		}
	}

	// watch the cluster-scoped resources we create, so we can undo unwnted modifications
	typesToWatch = []ctrlruntimeclient.Object{
		&admissionregistrationv1.MutatingWebhookConfiguration{},
		&admissionregistrationv1.ValidatingWebhookConfiguration{},
		&rbacv1.ClusterRoleBinding{},
		&kubermaticv1.AddonConfig{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, eventHandler, resources.ManagedByOperatorPredicate); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %w", t, err)
		}
	}

	// namespaces are not managed by the operator and so can use neither namespacePredicate
	// nor ManagedByPredicate, but still need to get their labels reconciled
	if err := c.Watch(&source.Kind{Type: &corev1.Namespace{}}, eventHandler, predicateutil.ByName(namespace)); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %w", &corev1.Namespace{}, err)
	}

	// The VPA gets resources deployed into the kube-system namespace.
	typesToWatch = []ctrlruntimeclient.Object{
		&appsv1.Deployment{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, eventHandler, predicateutil.ByNamespace(metav1.NamespaceSystem), resources.ManagedByOperatorPredicate); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %w", t, err)
		}
	}

	typesToWatch = []ctrlruntimeclient.Object{
		&rbacv1.ClusterRole{},
		&rbacv1.ClusterRoleBinding{},
	}

	for _, t := range typesToWatch {
		if err := c.Watch(&source.Kind{Type: t}, eventHandler, resources.ManagedByOperatorPredicate); err != nil {
			return fmt.Errorf("failed to create watcher for %T: %w", t, err)
		}
	}

	// CRDs are not owned by KKP, but still need to be updated accordingly
	if err := c.Watch(&source.Kind{Type: &apiextensionsv1.CustomResourceDefinition{}}, eventHandler, resources.ManagedByOperatorPredicate); err != nil {
		return fmt.Errorf("failed to create watcher for %T: %w", &apiextensionsv1.CustomResourceDefinition{}, err)
	}

	return nil
}
