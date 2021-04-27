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

package promtail

import (
	"k8c.io/kubermatic/v2/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ResourcesOnDeletion() []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.PromtailDaemonSetName,
				Namespace: resources.UserClusterMLANamespace,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.PromtailSecretName,
				Namespace: resources.UserClusterMLANamespace,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.PromtailCertificatesSecretName,
				Namespace: resources.MLANamespace,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resources.PromtailServiceAccountName,
				Namespace: resources.UserClusterMLANamespace,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.PromtailClusterRoleName,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: resources.PromtailClusterRoleBindingName,
			},
		},
	}
}
