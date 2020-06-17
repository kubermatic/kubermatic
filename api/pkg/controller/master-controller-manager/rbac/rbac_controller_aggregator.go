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

	"github.com/prometheus/client_golang/prometheus"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	k8scorev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog"
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
	rbacProjectController  *projectController
	rbacResourceController *resourcesController

	metrics             *Metrics
	allClusterProviders []*ClusterProvider
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

// New creates a new controller aggregator for managing RBAC for resources
func New(metrics *Metrics, allClusterProviders []*ClusterProvider, workerCount int) (*ControllerAggregator, error) {
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
				// do not reconcile secrets without "sa-token" and "credential" prefix
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
	}

	projectRBACCtrl, err := newProjectRBACController(metrics, allClusterProviders, projectResources)
	if err != nil {
		return nil, err
	}

	resourcesRBACCtrl, err := newResourcesController(metrics, allClusterProviders, projectResources)
	if err != nil {
		return nil, err
	}

	return &ControllerAggregator{
		workerCount:            workerCount,
		rbacProjectController:  projectRBACCtrl,
		rbacResourceController: resourcesRBACCtrl,
		metrics:                metrics,
		allClusterProviders:    allClusterProviders,
	}, nil
}

// Run starts the controller's worker routines. It is an implementation of
// sigs.k8s.io/controller-runtime/pkg/manager.Runnable
func (a *ControllerAggregator) Start(stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()

	// wait for all caches in all clusters to get in-sync
	for _, clusterProvider := range a.allClusterProviders {
		clusterProvider.StartInformers(stopCh)
		if err := clusterProvider.WaitForCachesToSync(stopCh); err != nil {
			return fmt.Errorf("failed to sync cache: %v", err)
		}
	}

	go a.rbacProjectController.run(a.workerCount, stopCh)
	go a.rbacResourceController.run(a.workerCount, stopCh)

	klog.Info("RBAC generator aggregator controller started")
	<-stopCh
	klog.Info("RBAC generator aggregator controller finished")

	return nil
}

func shouldEnqueueSecret(name string) bool {
	supportedPrefixes := []string{"sa-token", "credential"}
	for _, prefix := range supportedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
