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

	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	csiResourceName = "kubevirt-csi"
)

func csiServiceAccountReconciler(name, namespace string) reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return name, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			sa.Namespace = namespace
			return sa, nil
		}
	}
}

func csiSecretTokenReconciler(name string) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return name, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.SetAnnotations(map[string]string{
				corev1.ServiceAccountNameKey: csiResourceName,
			})

			s.Type = corev1.SecretTypeServiceAccountToken
			return s, nil
		}
	}
}

func csiRoleReconciler(name string) reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
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

func csiRoleBindingReconciler(name, namespace string) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
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
	roleReconcilers := []reconciling.NamedRoleReconcilerFactory{
		csiRoleReconciler(csiResourceName),
	}
	if err := reconciling.ReconcileRoles(ctx, roleReconcilers, namespace, client); err != nil {
		return err
	}

	roleBindingReconcilers := []reconciling.NamedRoleBindingReconcilerFactory{
		csiRoleBindingReconciler(csiResourceName, namespace),
	}
	if err := reconciling.ReconcileRoleBindings(ctx, roleBindingReconcilers, namespace, client); err != nil {
		return err
	}

	return nil
}

// reconcileInfraTokenAccess generates a service account token for KubeVirt CSI access.
func reconcileInfraTokenAccess(ctx context.Context, namespace string, client ctrlruntimeclient.Client) error {
	saReconcilers := []reconciling.NamedServiceAccountReconcilerFactory{
		csiServiceAccountReconciler(csiResourceName, namespace),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, saReconcilers, namespace, client); err != nil {
		return err
	}

	sa := corev1.ServiceAccount{}
	err := client.Get(ctx, types.NamespacedName{Name: csiResourceName, Namespace: namespace}, &sa)
	if err != nil {
		return err
	}

	if len(sa.Secrets) == 0 {
		// k8s 1.24 by default disabled automatic token creation for service accounts
		seReconcilers := []reconciling.NamedSecretReconcilerFactory{
			csiSecretTokenReconciler(csiResourceName),
		}
		if err := reconciling.ReconcileSecrets(ctx, seReconcilers, namespace, client); err != nil {
			return err
		}
	}

	return nil
}
