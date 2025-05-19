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

package userclusteradmissioncontrollerresources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Admission Controller
	admissionControllerServiceAccountName         = "kyverno-admission-controller"
	admissionControllerClusterRoleName            = "kyverno:admission-controller"
	admissionControllerCoreClusterRoleName        = "kyverno:admission-controller:core"
	admissionControllerClusterRoleBindingName     = "kyverno:admission-controller"
	admissionControllerViewClusterRoleBindingName = "kyverno:admission-controller:view"
)

// ClusterRoleReconciler returns the function to create and update the Kyverno admission controller cluster role.
func ClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return admissionControllerClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			cr.AggregationRule = &rbacv1.AggregationRule{
				ClusterRoleSelectors: []metav1.LabelSelector{
					{
						MatchLabels: map[string]string{
							"rbac.kyverno.io/aggregate-to-admission-controller": "true",
						},
					},
					{
						MatchLabels: map[string]string{
							"app.kubernetes.io/component": "admission-controller",
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

// CoreClusterRoleReconciler returns the function to create and update the Kyverno admission controller core cluster role.
func CoreClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return admissionControllerCoreClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
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
					APIGroups: []string{"admissionregistration.k8s.io"},
					Resources: []string{"mutatingwebhookconfigurations", "validatingwebhookconfigurations"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch", "deletecollection"},
				},
				{
					APIGroups: []string{"rbac.authorization.k8s.io"},
					Resources: []string{"roles", "clusterroles", "rolebindings", "clusterrolebindings"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{"policies", "policies/status", "clusterpolicies", "clusterpolicies/status",
						"updaterequests", "updaterequests/status", "globalcontextentries",
						"globalcontextentries/status", "policyexceptions"},
					Verbs: []string{"create", "delete", "get", "list", "patch", "update", "watch", "deletecollection"},
				},
				{
					APIGroups: []string{"policies.kyverno.io"},
					Resources: []string{"validatingpolicies", "validatingpolicies/status", "policyexceptions",
						"imagevalidatingpolicies", "imagevalidatingpolicies/status"},
					Verbs: []string{"create", "delete", "get", "list", "patch", "update", "watch", "deletecollection"},
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
					Verbs:     []string{"create", "update", "patch"},
				},
				{
					APIGroups: []string{"authorization.k8s.io"},
					Resources: []string{"subjectaccessreviews"},
					Verbs:     []string{"create"},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps", "namespaces"},
					Verbs:     []string{"get", "list", "watch"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"create", "update", "patch", "get", "list", "watch"},
				},
			}

			return cr, nil
		}
	}
}

// ClusterRoleBindingReconciler returns the function to create and update the Kyverno admission controller cluster role binding.
func ClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return admissionControllerClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
				"app.kubernetes.io/instance":  "kyverno",
				"app.kubernetes.io/part-of":   "kyverno",
				"app.kubernetes.io/version":   "v1.14.1",
			}

			crb.RoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     admissionControllerClusterRoleName,
			}

			crb.Subjects = []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      admissionControllerServiceAccountName,
					Namespace: "kyverno",
				},
			}

			return crb, nil
		}
	}
}

// ViewClusterRoleBindingReconciler returns the function to create and update the Kyverno admission controller view cluster role binding.
func ViewClusterRoleBindingReconciler() reconciling.NamedClusterRoleBindingReconcilerFactory {
	return func() (string, reconciling.ClusterRoleBindingReconciler) {
		return admissionControllerViewClusterRoleBindingName, func(crb *rbacv1.ClusterRoleBinding) (*rbacv1.ClusterRoleBinding, error) {
			crb.Labels = map[string]string{
				"app.kubernetes.io/component": "admission-controller",
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
					Name:      admissionControllerServiceAccountName,
					Namespace: "kyverno",
				},
			}

			return crb, nil
		}
	}
}
