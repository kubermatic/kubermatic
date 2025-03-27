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
	"context"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// Metrics contains metrics that this controller will collect and expose.
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

// ControllerAggregator type holds controllers for managing RBAC for projects and theirs resources.
type ControllerAggregator struct {
	workerCount             int
	rbacResourceControllers []*resourcesController

	metrics *Metrics
}

type projectResource struct {
	object      ctrlruntimeclient.Object
	destination string
	namespace   string

	// predicate is used by the controller-runtime to filter watched objects
	predicate func(o ctrlruntimeclient.Object) bool
}

// New creates a new controller aggregator for managing RBAC for resources.
func New(ctx context.Context, metrics *Metrics, mgr manager.Manager, seedManagerMap map[string]manager.Manager, log *zap.SugaredLogger, labelSelectorFunc func(*metav1.ListOptions), workerPredicate predicate.Predicate, workerCount int) (*ControllerAggregator, error) {
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
			object: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Secret",
				},
			},
			namespace: "kubermatic",
			predicate: func(o ctrlruntimeclient.Object) bool {
				// do not reconcile secrets without "sa-token", "credential" and "kubeconfig-external-cluster", "manifest-kubeone", "ssh-kubeone" prefix
				return shouldEnqueueSecret(o.GetName())
			},
		},
		{
			object: &kubermaticv1.User{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.UserKindName,
				},
			},
			predicate: func(o ctrlruntimeclient.Object) bool {
				return kubermaticv1helper.IsProjectServiceAccount(o.GetName())
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

		{
			object: &kubermaticv1.ClusterTemplateInstance{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ClusterTemplateInstanceKindName,
				},
			},
			destination: destinationSeed,
		},

		{
			object: &kubermaticv1.ResourceQuota{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ResourceQuotaKindName,
				},
			},
		},

		{
			object: &kubermaticv1.GroupProjectBinding{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.GroupProjectBindingKind,
				},
			},
		},
		{
			object: &kubermaticv1.ClusterBackupStorageLocation{
				TypeMeta: metav1.TypeMeta{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ClusterBackupStorageLocationKind,
				},
			},
			namespace: "kubermatic",
		},
	}

	if err := newProjectRBACController(ctx, metrics, mgr, seedManagerMap, log, projectResources, workerPredicate); err != nil {
		return nil, err
	}

	resourcesRBACCtrl, err := newResourcesControllers(ctx, metrics, mgr, log, seedManagerMap, projectResources)
	if err != nil {
		return nil, err
	}

	return &ControllerAggregator{
		workerCount:             workerCount,
		rbacResourceControllers: resourcesRBACCtrl,
		metrics:                 metrics,
	}, nil
}

func shouldEnqueueSecret(name string) bool {
	supportedPrefixes := []string{"sa-token", "credential", "kubeconfig-external-cluster", "manifest-kubeone", "ssh-kubeone"}
	for _, prefix := range supportedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}
