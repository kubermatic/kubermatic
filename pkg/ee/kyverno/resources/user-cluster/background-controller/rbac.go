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

package userclusterbackgroundcontrollerresources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Background Controller
	backgroundControllerServiceAccountName         = "kyverno-background-controller"
	backgroundControllerClusterRoleName            = "kyverno:background-controller"
	backgroundControllerCoreClusterRoleName        = "kyverno:background-controller:core"
	backgroundControllerClusterRoleBindingName     = "kyverno:background-controller"
	backgroundControllerViewClusterRoleBindingName = "kyverno:background-controller:view"
)

// BackgroundClusterRoleReconciler returns the function to create and update the Kyverno background controller cluster role.
func BackgroundClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return backgroundControllerClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			cr.AggregationRule = &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{
							"rbac.kyverno.io/aggregate-to-background-controller": "true",
						},
					},
					{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component": "background-controller",
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

// BackgroundCoreClusterRoleReconciler returns the function to create and update the Kyverno background controller core cluster role.
func BackgroundCoreClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return backgroundControllerCoreClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
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
					APIGroups: []string{"kyverno.io"},
					Resources: []string{"policies", "policies/status", "clusterpolicies", "clusterpolicies/status",
						"policyexceptions", "updaterequests", "updaterequests/status", "globalcontextentries",
						"globalcontextentries/status"},
					Verbs: []string{"create", "delete", "get", "list", "patch", "update", "watch", "deletecollection"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"namespaces", "configmaps"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"", "events.k8s.io"},
					Resources: []string{"events"},
					Verbs:     []string{"create", "get", "list", "patch", "update", "watch"},
				},
				{
					APIGroups: []string{"reports.kyverno.io"},
					Resources: []string{"ephemeralreports", "clusterephemeralreports"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch", "deletecollection"},
				},
				{
					APIGroups: []string{"networking.k8s.io"},
					Resources: []string{"ingresses", "ingressclasses", "networkpolicies"},
					Verbs:     []string{"create", "update", "patch", "delete"},
				},
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"rolebindings", "roles"},
					Verbs:     []string{"create", "update", "patch", "delete"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps", "resourcequotas", "limitranges"},
					Verbs:     []string{"create", "update", "patch", "delete"},
				},
			}

			return cr, nil
		}
	}
}

// BackgroundClusterRoleBindingReconciler returns the function to create and update the Kyverno background controller cluster role binding.
func BackgroundClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return backgroundControllerClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     backgroundControllerClusterRoleName,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      backgroundControllerServiceAccountName,
					Namespace: "kyverno",
				},
			}

			return crb, nil
		}
	}
}

// BackgroundViewClusterRoleBindingReconciler returns the function to create and update the Kyverno background controller view cluster role binding.
func BackgroundViewClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return backgroundControllerViewClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "background-controller",
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
					Name:      backgroundControllerServiceAccountName,
					Namespace: "kyverno",
				},
			}

			return crb, nil
		}
	}
}
