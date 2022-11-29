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

package operatingsystemmanager

import (
	"crypto/x509"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	webhookClusterRoleName        = "kubermatic:operating-system-manager-webhook"
	webhookClusterRoleBindingName = "kubermatic:operating-system-manager-webhook"

	clusterAPIGroup   = "cluster.k8s.io"
	clusterAPIVersion = "v1alpha1"
)

func WebhookClusterRoleCreator() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleCreator) {
		return webhookClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"operatingsystemmanager.k8c.io"},
					Resources: []string{"operatingsystemprofiles"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups:     []string{"coordination.k8s.io"},
					Resources:     []string{"leases"},
					Verbs:         []string{"create", "update", "get", "list", "watch"},
					ResourceNames: []string{"operating-system-manager-leader-lock"},
				},
			}

			return cr, nil
		}
	}
}

func WebhookClusterRoleBindingCreator() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return webhookClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.RoleRef = rbacv1.RoleRef{
				Name:     webhookClusterRoleName,
				Kind:     "ClusterRole",
				APIGroup: rbacv1.GroupName,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind:     rbacv1.UserKind,
					APIGroup: rbacv1.GroupName,
					Name:     resources.OperatingSystemManagerWebhookCertUsername,
				},
			}
			return crb, nil
		}
	}
}

// MutatingwebhookConfigurationCreator returns the MutatingwebhookConfiguration for OSM.
func MutatingwebhookConfigurationCreator(caCert *x509.Certificate, namespace string) reconciling.NamedMutatingWebhookConfigurationReconcilerFactory {
	return func() (string, reconciling.MutatingWebhookConfigurationCreator) {
		return resources.OperatingSystemManagerMutatingWebhookConfigurationName, func(mutatingWebhookConfiguration *admissionregistrationv1.MutatingWebhookConfiguration) (*admissionregistrationv1.MutatingWebhookConfiguration, error) {
			failurePolicy := admissionregistrationv1.Fail
			sideEffects := admissionregistrationv1.SideEffectClassNone
			mdURL := fmt.Sprintf("https://%s.%s.svc.cluster.local./mutate-v1alpha1-machinedeployment", resources.OperatingSystemManagerWebhookServiceName, namespace)
			reviewVersions := []string{"v1", "v1beta1"}

			// This only gets set when the APIServer supports it, so carry it over
			var scope *admissionregistrationv1.ScopeType
			if len(mutatingWebhookConfiguration.Webhooks) != 1 {
				mutatingWebhookConfiguration.Webhooks = []admissionregistrationv1.MutatingWebhook{{}}
			} else if len(mutatingWebhookConfiguration.Webhooks[0].Rules) > 0 {
				scope = mutatingWebhookConfiguration.Webhooks[0].Rules[0].Scope
			}

			mutatingWebhookConfiguration.Webhooks[0].Name = fmt.Sprintf("%s-machinedeployments", resources.OperatingSystemManagerMutatingWebhookConfigurationName)
			mutatingWebhookConfiguration.Webhooks[0].NamespaceSelector = &metav1.LabelSelector{}
			mutatingWebhookConfiguration.Webhooks[0].SideEffects = &sideEffects
			mutatingWebhookConfiguration.Webhooks[0].FailurePolicy = &failurePolicy
			mutatingWebhookConfiguration.Webhooks[0].AdmissionReviewVersions = reviewVersions
			mutatingWebhookConfiguration.Webhooks[0].Rules = []admissionregistrationv1.RuleWithOperations{{
				Operations: []admissionregistrationv1.OperationType{admissionregistrationv1.Create, admissionregistrationv1.Update},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{clusterAPIGroup},
					APIVersions: []string{clusterAPIVersion},
					Resources:   []string{"machinedeployments"},
					Scope:       scope,
				},
			}}
			mutatingWebhookConfiguration.Webhooks[0].ClientConfig = admissionregistrationv1.WebhookClientConfig{
				URL:      &mdURL,
				CABundle: triple.EncodeCertPEM(caCert),
			}

			return mutatingWebhookConfiguration, nil
		}
	}
}
