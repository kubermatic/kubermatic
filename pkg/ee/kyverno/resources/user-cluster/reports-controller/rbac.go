//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0")
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package userclusterreportscontrollerresources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	reportsControllerServiceAccountName         = "kyverno-reports-controller"
	reportsControllerClusterRoleName            = "kyverno:reports-controller"
	reportsControllerCoreClusterRoleName        = "kyverno:reports-controller:core"
	reportsControllerClusterRoleBindingName     = "kyverno:reports-controller"
	reportsControllerViewClusterRoleBindingName = "kyverno:reports-controller:view"
)

// ClusterRoleReconciler returns the function to create and update the Kyverno reports controller cluster role.
func ClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return reportsControllerClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "reports-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			cr.AggregationRule = &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{
							"rbac.kyverno.io/aggregate-to-reports-controller": "true",
						},
					},
					{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component": "reports-controller",
							"app.kubernetes.io/instance":  "kyverno",
							"app.kubernetes.io/part-of":   "kyverno",
						},
					},
				},
			}

			return cr, nil
		}
	}
}

// CoreClusterRoleReconciler returns the function to create and update the Kyverno reports controller core cluster role.
func CoreClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return reportsControllerCoreClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "reports-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"apiextensions.k8s.io"},
					Resources: []string{"customresourcedefinitions"},
					Verbs:     []string{"get"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps", "namespaces"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{"globalcontextentries", "globalcontextentries/status", "policyexceptions",
						"policies", "clusterpolicies"},
					Verbs: []string{"create", "delete", "get", "list", "patch", "update", "watch", "deletecollection"},
				},
				{
					APIGroups: []string{"policies.kyverno.io"},
					Resources: []string{"validatingpolicies", "validatingpolicies/status", "imagevalidatingpolicies",
						"imagevalidatingpolicies/status"},
					Verbs: []string{"create", "delete", "get", "list", "patch", "update", "watch", "deletecollection"},
				},
				{
					APIGroups: []string{"policies.kyverno.io"},
					Resources: []string{"policyexceptions", "policyexceptions/status"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"reports.kyverno.io"},
					Resources: []string{"ephemeralreports", "clusterephemeralreports"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch", "deletecollection"},
				},
				{
					APIGroups: []string{"wgpolicyk8s.io"},
					Resources: []string{"policyreports", "policyreports/status", "clusterpolicyreports", "clusterpolicyreports/status"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch", "deletecollection"},
				},
				{
					APIGroups: []string{"", "events.k8s.io"},
					Resources: []string{"events"},
					Verbs:     []string{"create", "patch"},
				},
			}

			return cr, nil
		}
	}
}

// ClusterRoleBindingReconciler returns the function to create and update the Kyverno reports controller cluster role binding.
func ClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return reportsControllerClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "reports-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     reportsControllerClusterRoleName,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      reportsControllerServiceAccountName,
					Namespace: "kyverno",
				},
			}

			return crb, nil
		}
	}
}

// ViewClusterRoleBindingReconciler returns the function to create and update the Kyverno reports controller view cluster role binding.
func ViewClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return reportsControllerViewClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "reports-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "view",
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      reportsControllerServiceAccountName,
					Namespace: "kyverno",
				},
			}

			return crb, nil
		}
	}
}
