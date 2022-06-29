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
	"fmt"
	"reflect"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func clusterRoleBindingCreator(binding kubermaticv1.GroupProjectBinding, clusterRole rbacv1.ClusterRole) reconciling.NamedClusterRoleBindingCreatorGetter {
	name := fmt.Sprintf("%s:%s", clusterRole.Name, binding.Name)
	return func() (string, reconciling.ClusterRoleBindingCreator) {
		return name, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			if crb.Labels == nil {
				crb.Labels = map[string]string{}
			}

			crb.Labels[kubermaticv1.AuthZGroupProjectBindingLabel] = binding.Name
			crb.Labels[kubermaticv1.AuthZRoleLabel] = binding.Spec.Role

			crb.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.GroupProjectBindingKind,
					Name:       binding.Name,
					UID:        binding.UID,
				},
			}
			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     clusterRole.Name,
			}
			crb.Subjects = []rbacv1.Subject{
				{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Group",
					Name:     binding.Spec.Group,
				},
			}

			return crb, nil
		}
	}
}

func roleBindingCreator(binding kubermaticv1.GroupProjectBinding, role rbacv1.Role) reconciling.NamedRoleBindingCreatorGetter {
	name := fmt.Sprintf("%s:%s", role.Name, binding.Name)
	return func() (string, reconciling.RoleBindingCreator) {
		return name, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			if rb.Labels == nil {
				rb.Labels = map[string]string{}
			}

			rb.Labels[kubermaticv1.AuthZGroupProjectBindingLabel] = binding.Name
			rb.Labels[kubermaticv1.AuthZRoleLabel] = binding.Spec.Role

			rb.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.GroupProjectBindingKind,
					Name:       binding.Name,
					UID:        binding.UID,
				},
			}
			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     role.Name,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Group",
					Name:     binding.Spec.Group,
				},
			}

			return rb, nil
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

		// generate patch and update the GroupProjectBinding
		return client.Patch(ctx, binding, ctrlruntimeclient.MergeFrom(original))
	})
}
