//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2021 Kubermatic GmbH

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

package resources

import (
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	serviceAccountName = "kubelb-ccm"
	roleName           = "kubelb-ccm"
	roleBindingName    = "kubelb-ccm"
)

func ServiceAccountReconciler() reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return serviceAccountName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

func RoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = resources.BaseAppLabels(resources.KubeLBAppName, nil)
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kubelb.k8c.io"},
					Resources: []string{"loadbalancers"},
					Verbs: []string{
						"create",
						"update",
						"get",
						"list",
						"delete",
						"patch",
						"watch",
					},
				},
				{
					APIGroups: []string{"kubelb.k8c.io"},
					Resources: []string{"loadbalancers/status"},
					Verbs: []string{
						"update",
						"get",
						"patch",
					},
				},
			}
			return r, nil
		}
	}
}

func RoleBindingReconciler(namespace string) reconciling.NamedRoleBindingReconcilerFactory {
	return func() (string, reconciling.RoleBindingReconciler) {
		return roleBindingName, func(rb *rbacv1.RoleBinding) (*rbacv1.RoleBinding, error) {
			rb.Labels = resources.BaseAppLabels(resources.KubeLBAppName, nil)

			rb.RoleRef = rbacv1.RoleRef{
				Name:     roleName,
				Kind:     "Role",
				APIGroup: rbacv1.GroupName,
			}
			rb.Subjects = []rbacv1.Subject{
				{
					Kind:      rbacv1.ServiceAccountKind,
					Name:      serviceAccountName,
					Namespace: namespace,
				},
			}
			return rb, nil
		}
	}
}
