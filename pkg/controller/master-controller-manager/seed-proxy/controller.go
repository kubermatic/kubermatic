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

package seedproxy

import (
	"context"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	predicateutil "k8c.io/kubermatic/v2/pkg/controller/util/predicate"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	// This controller is responsible for creating resources in master cluster required to proxy
	// components like prometheus from seed clusters.
	ControllerName = "kkp-seed-proxy-controller"

	// MasterDeploymentName is the name used for deployments'
	// NameLabel value.
	MasterDeploymentName = "seed-proxy"

	// MasterServiceName is the name used for services' NameLabel value.
	MasterServiceName = "seed-proxy"

	// MasterGrafanaNamespace is the namespace inside the master
	// cluster where Grafana is installed and where the ConfigMap
	// should be created in.
	MasterGrafanaNamespace = "monitoring-master"

	// MasterGrafanaConfigMapName is the name used for the newly
	// created Grafana ConfigMap.
	MasterGrafanaConfigMapName = "grafana-seed-proxies"

	// SeedServiceAccountName is the name used for service accounts
	// inside the seed cluster.
	SeedServiceAccountName = "seed-proxy"

	// SeedSecretName is the name used for service accounts
	// inside the seed cluster.
	SeedSecretName = "seed-proxy-token"

	// SeedMonitoringNamespace is the namespace inside the seed
	// cluster where Prometheus, Grafana etc. are installed.
	SeedMonitoringNamespace = "monitoring"

	// SeedPrometheusService is the service exposed by Prometheus.
	SeedPrometheusService = "prometheus:web"

	// SeedAlertmanagerService is the service exposed by Alertmanager.
	SeedAlertmanagerService = "alertmanager:http"

	// KubectlProxyPort is the port used by kubectl to provide the
	// proxy connection on. This is not the port on which any of the
	// target applications inside the seed (Prometheus, Grafana)
	// listen on.
	KubectlProxyPort = 8001

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

// Add creates a new Seed-Proxy controller that is responsible for
// establishing ServiceAccounts in all seeds and setting up proxy
// pods to allow access to monitoring applications inside the seed
// clusters, like Prometheus and Grafana.
func Add(
	mgr manager.Manager,
	numWorkers int,
	log *zap.SugaredLogger,
	namespace string,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter,
	configGetter provider.KubermaticConfigurationGetter,
) error {
	log = log.Named(ControllerName)

	reconciler := &Reconciler{
		Client:               mgr.GetClient(),
		recorder:             mgr.GetEventRecorderFor(ControllerName),
		log:                  log,
		namespace:            namespace,
		seedsGetter:          seedsGetter,
		seedKubeconfigGetter: seedKubeconfigGetter,
		seedClientGetter:     kubernetesprovider.SeedClientGetterFactory(seedKubeconfigGetter),
		configGetter:         configGetter,
	}

	bldr := builder.ControllerManagedBy(mgr).
		Named(ControllerName).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: numWorkers,
		})

	// watch seeds themselves
	namespacePredicate := predicateutil.ByNamespace(namespace)
	ownedPredicate := predicateutil.ByLabel(ManagedByLabel, ControllerName)

	bldr.For(&kubermaticv1.Seed{}, builder.WithPredicates(namespacePredicate))

	// watch related resources
	eventHandler := handler.EnqueueRequestsFromMapFunc(func(_ context.Context, a ctrlruntimeclient.Object) []reconcile.Request {
		seeds, err := seedsGetter()
		if err != nil {
			log.Errorw("failed to get seeds", zap.Error(err))
			return nil
		}

		var requests []reconcile.Request
		for seedName := range seeds {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: seedName},
			})
		}

		return requests
	})

	typesToWatch := []ctrlruntimeclient.Object{
		&appsv1.Deployment{},
		&corev1.Service{},
		&corev1.Secret{},
		&corev1.ConfigMap{},
	}

	for _, t := range typesToWatch {
		bldr.Watches(t, eventHandler, builder.WithPredicates(namespacePredicate, ownedPredicate))
	}

	_, err := bldr.Build(reconciler)

	return err
}
