//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package cleanupcontrollerresources

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	commonseedresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/common"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

// RoleReconciler returns the function to create and update the Kyverno cleanup controller role.
func RoleReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return commonseedresources.KyvernoCleanupControllerRoleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = commonseedresources.KyvernoLabels(commonseedresources.CleanupControllerComponentNameLabel)

			namespace := cluster.Status.NamespaceName
			tlsCaSecret := fmt.Sprintf("kyverno-cleanup-controller.%s.svc.kyverno-tls-ca", namespace)
			tlsPairSecret := fmt.Sprintf("kyverno-cleanup-controller.%s.svc.kyverno-tls-pair", namespace)

			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"secrets"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"secrets"},
					Verbs:         []string{"delete", "get", "list", "update", "watch"},
					ResourceNames: []string{tlsCaSecret, tlsPairSecret},
				},
				{
					APIGroups:     []string{""},
					Resources:     []string{"configmaps"},
					Verbs:         []string{"get", "list", "watch"},
					ResourceNames: []string{"kyverno", "kyverno-metrics"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups:     []string{"coordination.k8s.io"},
					Resources:     []string{"leases"},
					Verbs:         []string{"delete", "get", "patch", "update"},
					ResourceNames: []string{"kyverno-cleanup-controller"},
				},
				{
					APIGroups: []string{"apps"},
					Resources: []string{"deployments"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return r, nil
		}
	}
}

// RoleBindingReconciler returns the function to create and update the Kyverno cleanup controller role binding.
func RoleBindingReconciler(cluster *kubermaticv1.Cluster) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return commonseedresources.KyvernoCleanupControllerRoleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = commonseedresources.KyvernoLabels(commonseedresources.CleanupControllerComponentNameLabel)

			rb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     commonseedresources.KyvernoCleanupControllerRoleName,
			}

			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      commonseedresources.KyvernoCleanupControllerServiceAccountName,
					Namespace: cluster.Status.NamespaceName,
				},
			}

			return rb, nil
		}
	}
}
