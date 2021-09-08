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
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetResourcesToRemoveOnDelete() []ctrlruntimeclient.Object {
	var toRemove []ctrlruntimeclient.Object

	// Webhook
	toRemove = append(toRemove, &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperValidatingWebhookConfigurationName,
		}})
	toRemove = append(toRemove, &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperMutatingWebhookConfigurationName,
		}})

	// Pod Disruption Budget
	toRemove = append(toRemove, &policyv1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperPodDisruptionBudgetName,
			Namespace: resources.GatekeeperNamespace,
		}})

	// Deployment
	toRemove = append(toRemove, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperControllerDeploymentName,
			Namespace: resources.GatekeeperNamespace,
		},
	})
	toRemove = append(toRemove, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperAuditDeploymentName,
			Namespace: resources.GatekeeperNamespace,
		},
	})
	// Service
	toRemove = append(toRemove, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperWebhookServiceName,
			Namespace: resources.GatekeeperNamespace,
		},
	})
	// RBACs
	toRemove = append(toRemove, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperServiceAccountName,
			Namespace: resources.GatekeeperNamespace,
		},
	})
	toRemove = append(toRemove, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperRoleName,
			Namespace: resources.GatekeeperNamespace,
		},
	})
	toRemove = append(toRemove, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.GatekeeperRoleBindingName,
			Namespace: resources.GatekeeperNamespace,
		},
	})
	toRemove = append(toRemove, &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperRoleName,
		},
	})
	toRemove = append(toRemove, &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperRoleBindingName,
		},
	})

	// CRDs
	toRemove = append(toRemove, &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperConstraintTemplateCRDName,
		}})
	toRemove = append(toRemove, &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperConfigCRDName,
		}})
	toRemove = append(toRemove, &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperConstraintPodStatusCRDName,
		}})
	toRemove = append(toRemove, &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperConstraintTemplatePodStatusCRDName,
		}})
	toRemove = append(toRemove, &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperMutatorPodStatusCRDName,
		}})
	toRemove = append(toRemove, &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperAssignCRDName,
		}})
	toRemove = append(toRemove, &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperAssignMetadataCRDName,
		}})
	// Namespace
	toRemove = append(toRemove, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: resources.GatekeeperNamespace,
		}})

	return toRemove
}
