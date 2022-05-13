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

package resources

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func EnsureAllDeleted(ctx context.Context, client ctrlruntimeclient.Client) error {
	objects := []ctrlruntimeclient.Object{
		&appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      AgentDaemonSetName,
				Namespace: metav1.NamespaceSystem,
			},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      OperatorDeploymentName,
				Namespace: metav1.NamespaceSystem,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: OperatorClusterRoleBindingName,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: AgentClusterRoleBindingName,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: OperatorClusterRoleName,
			},
		},
		&rbacv1.ClusterRole{
			ObjectMeta: metav1.ObjectMeta{
				Name: AgentClusterRoleName,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      OperatorServiceAccountName,
				Namespace: metav1.NamespaceSystem,
			},
		},
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      AgentServiceAccountName,
				Namespace: metav1.NamespaceSystem,
			},
		},
	}

	for _, object := range objects {
		if err := ensureDeleted(ctx, client, object); err != nil {
			return err
		}
	}

	return nil
}

func ensureDeleted(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object) error {
	if err := client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		// error is 'not found', we're done here
		return nil
	}

	return client.Delete(ctx, obj)
}
