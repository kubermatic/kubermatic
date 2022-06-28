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

package grouprbac

import (
	"context"
	"reflect"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func clusterRoleBindingCreator(binding *kubermaticv1.GroupProjectBinding, clusterRole *rbacv1.ClusterRole) reconciling.NamedClusterRoleBindingCreatorGetter {
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return "test", func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.GroupProjectBindingKind,
					Name:       binding.Name,
				},
			}
			crb.RoleRef = rbacv1.RoleRef{
				Kind: "ClusterRole",
				Name: clusterRole.Name,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					Kind: "Group",
					Name: binding.Spec.Group,
				},
			}

			return &rbacv1.ClusterRoleBinding{}, nil
		}
	}
}

type GroupProjectBindingPatchFunc func(binding *kubermaticv1.GroupProjectBinding)

func updateGroupProjectBinding(ctx context.Context, client ctrlruntimeclient.Client, binding *kubermaticv1.GroupProjectBinding, patch GroupProjectBindingPatchFunc) error {
	key := ctrlruntimeclient.ObjectKeyFromObject(binding)

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		// fetch the current state of the binding
		if err := client.Get(ctx, key, binding); err != nil {
			return err
		}

		// modify it
		original := binding.DeepCopy()
		patch(binding)

		// save some work
		if reflect.DeepEqual(original, binding) {
			return nil
		}

		// update the status
		return client.Patch(ctx, binding, ctrlruntimeclient.MergeFrom(original))
	})
}
