/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubernetesdashboard

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func ResourcesForDeletion(namespace string) []ctrlruntimeclient.Object {
	return []ctrlruntimeclient.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: apiDeploymentName, Namespace: namespace}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: authDeploymentName, Namespace: namespace}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: kongDeploymentName, Namespace: namespace}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: webDeploymentName, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: apiServiceName, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: authServiceName, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: kongServiceName, Namespace: namespace}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: webServiceName, Namespace: namespace}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: KongConfigMapName, Namespace: namespace}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: kongServiceAccountName, Namespace: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: CSRFSecretName, Namespace: namespace}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: KubeconfigSecretName, Namespace: namespace}},
	}
}
