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

package userclustercommonresources

import (
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	// Admin roles
	adminPoliciesClusterRoleName       = "kyverno:rbac:admin:policies"
	adminPolicyReportsClusterRoleName  = "kyverno:rbac:admin:policyreports"
	adminReportsClusterRoleName        = "kyverno:rbac:admin:reports"
	adminUpdateRequestsClusterRoleName = "kyverno:rbac:admin:updaterequests"

	// View roles
	viewPoliciesClusterRoleName       = "kyverno:rbac:view:policies"
	viewPolicyReportsClusterRoleName  = "kyverno:rbac:view:policyreports"
	viewReportsClusterRoleName        = "kyverno:rbac:view:reports"
	viewUpdateRequestsClusterRoleName = "kyverno:rbac:view:updaterequests"
)

// AdminPoliciesClusterRoleReconciler returns the function to create and update the Kyverno admin policies cluster role.
func AdminPoliciesClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return adminPoliciesClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component":                  "rbac",
				"app.kubernetes.io/instance":                   "kyverno",
				"app.kubernetes.io/part-of":                    "kyverno",
				"app.kubernetes.io/version":                    "v1.14.1",
				"rbac.authorization.k8s.io/aggregate-to-admin": "true",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{"cleanuppolicies", "clustercleanuppolicies", "policies", "clusterpolicies"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
				},
			}

			return cr, nil
		}
	}
}

// ViewPoliciesClusterRoleReconciler returns the function to create and update the Kyverno view policies cluster role.
func ViewPoliciesClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return viewPoliciesClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component":                 "rbac",
				"app.kubernetes.io/instance":                  "kyverno",
				"app.kubernetes.io/part-of":                   "kyverno",
				"app.kubernetes.io/version":                   "v1.14.1",
				"rbac.authorization.k8s.io/aggregate-to-view": "true",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{"cleanuppolicies", "clustercleanuppolicies", "policies", "clusterpolicies"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return cr, nil
		}
	}
}

// AdminPolicyReportsClusterRoleReconciler returns the function to create and update the Kyverno admin policy reports cluster role.
func AdminPolicyReportsClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return adminPolicyReportsClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component":                  "rbac",
				"app.kubernetes.io/instance":                   "kyverno",
				"app.kubernetes.io/part-of":                    "kyverno",
				"app.kubernetes.io/version":                    "v1.14.1",
				"rbac.authorization.k8s.io/aggregate-to-admin": "true",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"wgpolicyk8s.io"},
					Resources: []string{"policyreports", "clusterpolicyreports"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
				},
			}

			return cr, nil
		}
	}
}

// ViewPolicyReportsClusterRoleReconciler returns the function to create and update the Kyverno view policy reports cluster role.
func ViewPolicyReportsClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return viewPolicyReportsClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component":                 "rbac",
				"app.kubernetes.io/instance":                  "kyverno",
				"app.kubernetes.io/part-of":                   "kyverno",
				"app.kubernetes.io/version":                   "v1.14.1",
				"rbac.authorization.k8s.io/aggregate-to-view": "true",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"wgpolicyk8s.io"},
					Resources: []string{"policyreports", "clusterpolicyreports"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return cr, nil
		}
	}
}

// AdminReportsClusterRoleReconciler returns the function to create and update the Kyverno admin reports cluster role.
func AdminReportsClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return adminReportsClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component":                  "rbac",
				"app.kubernetes.io/instance":                   "kyverno",
				"app.kubernetes.io/part-of":                    "kyverno",
				"app.kubernetes.io/version":                    "v1.14.1",
				"rbac.authorization.k8s.io/aggregate-to-admin": "true",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"reports.kyverno.io"},
					Resources: []string{"ephemeralreports", "clusterephemeralreports"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
				},
			}

			return cr, nil
		}
	}
}

// ViewReportsClusterRoleReconciler returns the function to create and update the Kyverno view reports cluster role.
func ViewReportsClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return viewReportsClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component":                 "rbac",
				"app.kubernetes.io/instance":                  "kyverno",
				"app.kubernetes.io/part-of":                   "kyverno",
				"app.kubernetes.io/version":                   "v1.14.1",
				"rbac.authorization.k8s.io/aggregate-to-view": "true",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"reports.kyverno.io"},
					Resources: []string{"ephemeralreports", "clusterephemeralreports"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return cr, nil
		}
	}
}

// AdminUpdateRequestsClusterRoleReconciler returns the function to create and update the Kyverno admin update requests cluster role.
func AdminUpdateRequestsClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return adminUpdateRequestsClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component":                  "rbac",
				"app.kubernetes.io/instance":                   "kyverno",
				"app.kubernetes.io/part-of":                    "kyverno",
				"app.kubernetes.io/version":                    "v1.14.1",
				"rbac.authorization.k8s.io/aggregate-to-admin": "true",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{"updaterequests"},
					Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
				},
			}

			return cr, nil
		}
	}
}

// ViewUpdateRequestsClusterRoleReconciler returns the function to create and update the Kyverno view update requests cluster role.
func ViewUpdateRequestsClusterRoleReconciler() reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return viewUpdateRequestsClusterRoleName, func(cr *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
			cr.Labels = map[string]string{
				"app.kubernetes.io/component":                 "rbac",
				"app.kubernetes.io/instance":                  "kyverno",
				"app.kubernetes.io/part-of":                   "kyverno",
				"app.kubernetes.io/version":                   "v1.14.1",
				"rbac.authorization.k8s.io/aggregate-to-view": "true",
			}

			cr.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{"kyverno.io"},
					Resources: []string{"updaterequests"},
					Verbs:     []string{"get", "list", "watch"},
				},
			}

			return cr, nil
		}
	}
}
