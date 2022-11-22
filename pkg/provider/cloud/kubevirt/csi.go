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

package kubevirt

import (
	"context"

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	csiResourceName = "kubevirt-csi"
)

func csiServiceAccountCreator(name, namespace string) reconciling.NamedServiceAccountCreatorGetter {
	return func() (string, reconciling.ServiceAccountCreator) {
		return name, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Namespace = namespace
			return sa, nil
		}
	}
}

func csiSecretTokenCreator(name string) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return name, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.SetAnnotations(map[string]string{
				corev1.ServiceAccountNameKey: csiResourceName,
			})

			s.Type = corev1.SecretTypeServiceAccountToken
			return s, nil
		}
	}
}

func csiRoleCreator(name string) reconciling.NamedRoleCreatorGetter {
	return func() (string, reconciling.RoleCreator) {
		return name, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"cdi.kubevirt.io"},
					Resources: []string{"datavolumes"},
					Verbs:     []string{"get", "create", "delete"},
				},
				{
					APIGroups: []string{"kubevirt.io"},
					Resources: []string{"virtualmachineinstances"},
					Verbs:     []string{"list"},
				},
				{
					APIGroups: []string{"subresources.kubevirt.io"},
					Resources: []string{"virtualmachineinstances/addvolume", "virtualmachineinstances/removevolume"},
					Verbs:     []string{"update"},
				},
			}

			return r, nil
		}
	}
}

func csiRoleBindingCreator(name, namespace string) reconciling.NamedRoleBindingCreatorGetter {
	return func() (string, reconciling.RoleBindingCreator) {
		return name, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      name,
					Namespace: namespace,
				},
			}

			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     name,
			}

			return rb, nil
		}
	}
}

// reconcileCSIRoleRoleBinding reconciles the Role and RoleBinding needed by CSI driver.
func reconcileCSIRoleRoleBinding(ctx context.Context, namespace string, client ctrlruntimeclient.Client) error {
	roleCreators := []reconciling.NamedRoleCreatorGetter{
		csiRoleCreator(csiResourceName),
	}
	if err := reconciling.ReconcileRoles(ctx, roleCreators, namespace, client); err != nil {
		return err
	}

	roleBindingCreators := []reconciling.NamedRoleBindingCreatorGetter{
		csiRoleBindingCreator(csiResourceName, namespace),
	}
	if err := reconciling.ReconcileRoleBindings(ctx, roleBindingCreators, namespace, client); err != nil {
		return err
	}

	return nil
}

// reconcileInfraTokenAccess generates a service account token for KubeVirt CSI access.
func reconcileInfraTokenAccess(ctx context.Context, namespace string, client ctrlruntimeclient.Client) error {
	saCreators := []reconciling.NamedServiceAccountCreatorGetter{
		csiServiceAccountCreator(csiResourceName, namespace),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, saCreators, namespace, client); err != nil {
		return err
	}

	sa := corev1.ServiceAccount{}
	err := client.Get(ctx, types.NamespacedName{Name: csiResourceName, Namespace: namespace}, &sa)
	if err != nil {
		return err
	}

	if len(sa.Secrets) == 0 {
		// k8s 1.24 by default disabled automatic token creation for service accounts
		seCreators := []reconciling.NamedSecretCreatorGetter{
			csiSecretTokenCreator(csiResourceName),
		}
		if err := reconciling.ReconcileSecrets(ctx, seCreators, namespace, client); err != nil {
			return err
		}
	}

	return nil
}
