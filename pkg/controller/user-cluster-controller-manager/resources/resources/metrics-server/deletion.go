/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package metricsserver

import (
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UserClusterResourcesForDeletion contains a set of resources deployed in user
// cluster if metrics-server is running fully in the user cluster (not in seed).
// It does not cover common metrics-server resources that are being deployed
// regardless of the deployment strategy (in seed / in user cluster).
func UserClusterResourcesForDeletion() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.MetricsServerDeploymentName,
				Namespace: metav1.NamespaceSystem,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      servingCertSecretName,
				Namespace: metav1.NamespaceSystem,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.MetricsServerServiceAccountName,
				Namespace: metav1.NamespaceSystem,
			},
		},
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.MetricsServerPodDisruptionBudgetName,
				Namespace: metav1.NamespaceSystem,
			},
		},
	}
}
