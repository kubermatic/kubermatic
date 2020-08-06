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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
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
	workerCount            int
	rbacResourceController *resourcesController

	metrics               *Metrics
	masterClusterProvider *ClusterProvider
	seedClusterProviders  []*ClusterProvider
}

type projectResource struct {
	gvr         schema.GroupVersionResource
	kind        string
	destination string
	namespace   string

	// shouldEnqueue is a convenience function that is called right before
	// the object is added to the queue. This is your last chance to say "no"
	shouldEnqueue func(obj metav1.Object) bool
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
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.ClusterResourceName,
			},
			kind:        kubermaticv1.ClusterKindName,
			destination: destinationSeed,
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.SSHKeyResourceName,
			},
			kind: kubermaticv1.SSHKeyKind,
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.UserProjectBindingResourceName,
			},
			kind: kubermaticv1.UserProjectBindingKind,
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    k8scorev1.GroupName,
				Version:  k8scorev1.SchemeGroupVersion.Version,
				Resource: "secrets",
			},
			kind:      "Secret",
			namespace: "kubermatic",
			shouldEnqueue: func(obj metav1.Object) bool {
				// do not reconcile secrets without "sa-token", "credential" and "kubeconfig-external-cluster" prefix
				return shouldEnqueueSecret(obj.GetName())
			},
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.UserResourceName,
			},
			kind: kubermaticv1.UserKindName,
			shouldEnqueue: func(obj metav1.Object) bool {
				// do not reconcile resources without "serviceaccount" prefix
				return strings.HasPrefix(obj.GetName(), "serviceaccount")
			},
		},

		{
			gvr: schema.GroupVersionResource{
				Group:    kubermaticv1.GroupName,
				Version:  kubermaticv1.GroupVersion,
				Resource: kubermaticv1.ExternalClusterResourceName,
			},
			kind: kubermaticv1.ExternalClusterKind,
		},
	}

	err = newProjectRBACController(metrics, mgr, seedManagerMap, masterClusterProvider, projectResources, workerPredicate)
	if err != nil {
		return nil, err
	}

	resourcesRBACCtrl, err := newResourcesController(metrics, masterClusterProvider, seedClusterProviders, projectResources)
	if err != nil {
		return nil, err
	}

	return &ControllerAggregator{
		workerCount:            workerCount,
		rbacResourceController: resourcesRBACCtrl,
		metrics:                metrics,
		masterClusterProvider:  masterClusterProvider,
		seedClusterProviders:   seedClusterProviders,
	}, nil
}

// Run starts the controller's worker routines. It is an implementation of
// sigs.k8s.io/controller-runtime/pkg/manager.Runnable
func (a *ControllerAggregator) Start(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	// wait for all caches in all clusters to get in-sync
	for _, clusterProvider := range append(a.seedClusterProviders, a.masterClusterProvider) {
		clusterProvider.StartInformers(stopCh)
		if err := clusterProvider.WaitForCachesToSync(stopCh); err != nil {
			return fmt.Errorf("failed to sync cache: %v", err)
		}
	}

	go a.rbacResourceController.run(a.workerCount, stopCh)

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
