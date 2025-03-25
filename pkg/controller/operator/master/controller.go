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

package master

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/workerlabel"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller is responsible for creating/updating/deleting all the required resources on the master clusters.
	ControllerName = "kkp-master-operator"
)

func Add(
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
		versions:   kubermatic.GetVersions(),
	}

	bldr := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		})

	namespacePredicate := predicateutil.ByNamespace(namespace)
	workerNamePredicate := workerlabel.Predicate(workerName)

	// put the config's identifier on the queue
	kubermaticConfigHandler := handler.EnqueueRequestsFromMapFunc(func(_ context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		return []reconcile.Request{
			{
				NamespacedName: types.NamespacedName{
					Namespace: a.GetNamespace(),
					Name:      a.GetName(),
				},
			},
		}
	})

	bldr.Watches(&kubermaticv1.KubermaticConfiguration{}, kubermaticConfigHandler, builder.WithPredicates(namespacePredicate, workerNamePredicate))

	// for each child put the parent configuration onto the queue
	childEventHandler := handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		config, err := kubernetes.GetRawKubermaticConfiguration(ctx, mgr.GetClient(), namespace)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to get KubermaticConfiguration: %w", err))
			return nil
		}

		// when handling namespaces, it's okay to not find a KubermaticConfiguration
		// and simply skip reconciling
		if errors.Is(err, provider.ErrNoKubermaticConfigurationFound) {
			return nil
		}

		if errors.Is(err, provider.ErrTooManyKubermaticConfigurationFound) {
			log.Warnw("found multiple KubermaticConfigurations in this namespace, refusing to guess the owner", "namespace", namespace)
			return nil
		}

		if config.Labels[kubermaticv1.WorkerNameLabelKey] != workerName {
			log.Debugf("KubermaticConfiguration does not have matching %s label", kubermaticv1.WorkerNameLabelKey)
			return nil
		}

		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Namespace: config.Namespace,
				Name:      config.Name,
			},
		}}
	})

	for _, t := range []ctrlruntimeclient.Object{
		&appsv1.Deployment{},
		&corev1.ConfigMap{},
		&corev1.Secret{},
		&corev1.Service{},
		&corev1.ServiceAccount{},
		&networkingv1.Ingress{},
		&policyv1.PodDisruptionBudget{},
	} {
		bldr.Watches(t, childEventHandler, builder.WithPredicates(namespacePredicate, common.ManagedByOperatorPredicate))
	}

	for _, t := range []ctrlruntimeclient.Object{
		&admissionregistrationv1.ValidatingWebhookConfiguration{},
		&rbacv1.ClusterRoleBinding{},
	} {
		bldr.Watches(t, childEventHandler, builder.WithPredicates(common.ManagedByOperatorPredicate))
	}

	for _, t := range []ctrlruntimeclient.Object{
		&kubermaticv1.AddonConfig{},
	} {
		bldr.Watches(t, childEventHandler)
	}

	// namespaces are not managed by the operator and so can use neither namespacePredicate
	// nor ManagedByPredicate, but still need to get their labels reconciled
	bldr.Watches(&corev1.Namespace{}, childEventHandler, builder.WithPredicates(predicateutil.ByName(namespace)))

	_, err := bldr.Build(reconciler)

	return err
}
