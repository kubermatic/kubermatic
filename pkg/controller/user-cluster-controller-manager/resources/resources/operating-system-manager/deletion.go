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
	"k8c.io/kubermatic/v2/pkg/resources"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ResourcesForDeletion() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		// RBAC
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.OperatingSystemManagerRoleName,
				Namespace: metav1.NamespaceSystem,
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.OperatingSystemManagerRoleName,
				Namespace: metav1.NamespacePublic,
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.OperatingSystemManagerRoleName,
				Namespace: metav1.NamespaceDefault,
			},
		},
		&rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.OperatingSystemManagerRoleName,
				Namespace: resources.CloudInitSettingsNamespace,
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.OperatingSystemManagerRoleBindingName,
				Namespace: metav1.NamespaceSystem,
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.OperatingSystemManagerRoleBindingName,
				Namespace: metav1.NamespacePublic,
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.OperatingSystemManagerRoleBindingName,
				Namespace: metav1.NamespaceDefault,
			},
		},
		&rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.OperatingSystemManagerRoleBindingName,
				Namespace: resources.CloudInitSettingsNamespace,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.OperatingSystemManagerClusterRoleName,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.OperatingSystemManagerClusterRoleBindingName,
			},
		},
		// Webhooks
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: webhookClusterRoleName,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: webhookClusterRoleBindingName,
			},
		},
		&admissionregistrationv1.MutatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.OperatingSystemManagerMutatingWebhookConfigurationName,
			},
		},
		&admissionregistrationv1.ValidatingWebhookConfiguration{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.OperatingSystemManagerValidatingWebhookConfigurationName,
			},
		},
		// CRDs
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.OperatingSystemManagerOperatingSystemProfileCRDName,
			},
		},
		&apiextensionsv1.CustomResourceDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.OperatingSystemManagerOperatingSystemConfigCRDName,
			},
		},
	}
}
