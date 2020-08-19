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

package rbac

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	kubermaticclientset "k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned"
	"k8c.io/kubermatic/v2/pkg/crd/client/informers/externalversions"
	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	util "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Metrics contains metrics that this controller will collect and expose
type Metrics struct {
	Workers prometheus.Gauge
}

// NewMetrics creates RBACGeneratorControllerMetrics
// with default values initialized, so metrics always show up.
func NewMetrics() *Metrics {
	subsystem := "rbac_generator_controller"
	cm := &Metrics{
		Workers: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running RBACGenerator controller workers",
		}),
	}

	cm.Workers.Set(0)
	return cm
}

// ControllerAggregator type holds controllers for managing RBAC for projects and theirs resources
type ControllerAggregator struct {
	workerCount             int
	rbacResourceControllers []*resourcesController

	metrics               *Metrics
	masterClusterProvider *ClusterProvider
	seedClusterProviders  []*ClusterProvider
}

type projectResource struct {
	object      runtime.Object
	destination string
	namespace   string

	// predicate is used by the controller-runtime to filter watched objects
	predicate func(m metav1.Object, r runtime.Object) bool
}

func restConfigToInformer(cfg *rest.Config, name string, labelSelectorFunc func(*metav1.ListOptions)) (*ClusterProvider, error) {
	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubeClient: %v", err)
	}
	kubermaticClient, err := kubermaticclientset.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubermaticClient: %v", err)
	}
	kubermaticInformerFactory := externalversions.NewFilteredSharedInformerFactory(kubermaticClient, time.Minute*5, metav1.NamespaceAll, labelSelectorFunc)
	kubeInformerProvider := NewInformerProvider(kubeClient, time.Minute*5)

	return NewClusterProvider(name, kubeClient, kubeInformerProvider, kubermaticClient, kubermaticInformerFactory), nil
}

func managersToInformers(mgr manager.Manager, seedManagerMap map[string]manager.Manager, selectorOps func(*metav1.ListOptions)) (*ClusterProvider, []*ClusterProvider, error) {
	seedClusterProviders := []*ClusterProvider{}

	for seedName, seedMgr := range seedManagerMap {
		clusterProvider, err := restConfigToInformer(seedMgr.GetConfig(), seedName, selectorOps)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create rbac provider for seed %q: %v", seedName, err)
		}
		seedClusterProviders = append(seedClusterProviders, clusterProvider)
	}

	masterClusterProvider, err := restConfigToInformer(mgr.GetConfig(), "master", selectorOps)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create master rbac provider: %v", err)
	}

	return masterClusterProvider, seedClusterProviders, nil
}

// New creates a new controller aggregator for managing RBAC for resources
func New(metrics *Metrics, mgr manager.Manager, seedManagerMap map[string]manager.Manager, labelSelectorFunc func(*metav1.ListOptions), workerPredicate predicate.Predicate, workerCount int) (*ControllerAggregator, error) {
	// Convert the controller-runtime's managers to old-school informers.
	masterClusterProvider, seedClusterProviders, err := managersToInformers(mgr, seedManagerMap, labelSelectorFunc)
	if err != nil {
		return nil, err
	}

	projectResources := []projectResource{
		{
			object: &kubermaticv1.Cluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ClusterKindName,
				},
			},
			destination: destinationSeed,
		},

		{
			object: &kubermaticv1.UserSSHKey{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.SSHKeyKind,
				},
			},
		},

		{
			object: &kubermaticv1.UserProjectBinding{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserProjectBindingKind,
				},
			},
		},

		{
			object: &k8scorev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
			},
			namespace: "kubermatic",
			predicate: func(m metav1.Object, r runtime.Object) bool {
				// do not reconcile secrets without "sa-token", "credential" and "kubeconfig-external-cluster" prefix
				return shouldEnqueueSecret(m.GetName())
			},
		},
		{
			object: &kubermaticv1.User{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
				},
			},
			predicate: func(m metav1.Object, r runtime.Object) bool {
				// do not reconcile resources without "serviceaccount" prefix
				return strings.HasPrefix(m.GetName(), "serviceaccount")
			},
		},

		{
			object: &kubermaticv1.ExternalCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ExternalClusterKind,
				},
			},
		},
	}

	err = newProjectRBACController(metrics, mgr, seedManagerMap, masterClusterProvider, projectResources, workerPredicate)
	if err != nil {
		return nil, err
	}

	resourcesRBACCtrl, err := newResourcesControllers(metrics, mgr, seedManagerMap, masterClusterProvider, seedClusterProviders, projectResources)
	if err != nil {
		return nil, err
	}

	return &ControllerAggregator{
		workerCount:             workerCount,
		rbacResourceControllers: resourcesRBACCtrl,
		metrics:                 metrics,
		masterClusterProvider:   masterClusterProvider,
		seedClusterProviders:    seedClusterProviders,
	}, nil
}

// Start starts the controller's worker routines. It is an implementation of
// sigs.k8s.io/controller-runtime/pkg/manager.Runnable
func (a *ControllerAggregator) Start(stopCh <-chan struct{}) error {
	defer util.HandleCrash()

	// wait for all caches in all clusters to get in-sync
	for _, clusterProvider := range append(a.seedClusterProviders, a.masterClusterProvider) {
		clusterProvider.StartInformers(stopCh)
		if err := clusterProvider.WaitForCachesToSync(stopCh); err != nil {
			return fmt.Errorf("failed to sync cache: %v", err)
		}
	}

	for _, ctl := range a.rbacResourceControllers {
		go ctl.run(a.workerCount, stopCh)
	}

	klog.Info("RBAC generator aggregator controller started")
	<-stopCh
	klog.Info("RBAC generator aggregator controller finished")

	return nil
}

func shouldEnqueueSecret(name string) bool {
	supportedPrefixes := []string{"sa-token", "credential", "kubeconfig-external-cluster"}
	for _, prefix := range supportedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
