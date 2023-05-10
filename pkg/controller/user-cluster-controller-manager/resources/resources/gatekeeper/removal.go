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

package gatekeeper

import (
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetResourcesToRemoveOnDelete() ([]ctrlruntimeclient.Object, error) {
	var toRemove []ctrlruntimeclient.Object

	toRemove = append(toRemove,
		// Webhooks
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.GatekeeperValidatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.GatekeeperMutatingWebhookConfigurationName,
			},
		},

		// Pod Disruption Budget
		&policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.GatekeeperPodDisruptionBudgetName,
				Namespace: resources.GatekeeperNamespace,
			},
		},

		// Deployments
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.GatekeeperControllerDeploymentName,
				Namespace: resources.GatekeeperNamespace,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.GatekeeperAuditDeploymentName,
				Namespace: resources.GatekeeperNamespace,
			},
		},

		// Service
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.GatekeeperWebhookServiceName,
				Namespace: resources.GatekeeperNamespace,
			},
		},

		// RBAC
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.GatekeeperServiceAccountName,
				Namespace: resources.GatekeeperNamespace,
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.GatekeeperRoleName,
				Namespace: resources.GatekeeperNamespace,
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.GatekeeperRoleBindingName,
				Namespace: resources.GatekeeperNamespace,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.GatekeeperRoleName,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.GatekeeperRoleBindingName,
			},
		},
	)

	// CRDs
	crds, err := CRDs()
	if err != nil {
		return nil, err
	}

	for _, crd := range crds {
		toRemove = append(toRemove, &apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: crd.Name,
			},
		})
	}

	// Namespace
	toRemove = append(toRemove,
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.GatekeeperNamespace,
			},
		},
	)

	return toRemove, nil
}
