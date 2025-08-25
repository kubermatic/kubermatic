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
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	rbacv1 "k8s.io/api/rbac/v1"
)

const (
	roleName               = "kubermatic-kubelb-ccm"
	roleBindingName        = "kubermatic-kubelb-ccm"
	clusterRoleName        = "system:kubermatic-kubelb-ccm"
	clusterRoleBindingName = "system:kubermatic-kubelb-ccm"
)

// KubeSystemRoleReconciler returns the func to create/update the Role for leader election.
func KubeSystemRoleReconciler() reconciling.NamedRoleReconcilerFactory {
	return func() (string, reconciling.RoleReconciler) {
		return roleName, func(r *rbacv1.Role) (*rbacv1.Role, error) {
			r.Labels = resources.BaseAppLabels(resources.KubeLBAppName, nil)
			r.Rules = []rbacv1.PolicyRule{
				{
					APIGroups: []string{""},
					Resources: []string{"configmaps"},
					Verbs: []string{
						"create",
						"update",
						"get",
						"list",
					},
				},
				{
					APIGroups: []string{""},
					Resources: []string{"events"},
					Verbs: []string{
						"create",
						"patch",
					},
				},
				{
					APIGroups:     []string{"coordination.k8s.io"},
					Resources:     []string{"leases"},
					Verbs:         []string{"get", "update", "patch"},
					ResourceNames: []string{"19f32e7b.ccm.kubelb.k8c.io"},
				},
				{
					APIGroups: []string{"coordination.k8s.io"},
					Resources: []string{"leases"},
					Verbs:     []string{"create"},
				},
			}
			return r, nil
		}
	}
}

func ClusterRoleReconciler(dc kubermaticv1.Datacenter, cluster *kubermaticv1.Cluster) reconciling.NamedClusterRoleReconcilerFactory {
	return func() (string, reconciling.ClusterRoleReconciler) {
		return clusterRoleName,
			func(r *rbacv1.ClusterRole) (*rbacv1.ClusterRole, error) {
				r.Rules = []rbacv1.PolicyRule{
					{
						APIGroups: []string{""},
						Resources: []string{"nodes"},
						Verbs:     []string{"get", "list", "watch"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"services"},
						Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
					},
					{
						APIGroups: []string{""},
						Resources: []string{"services/status"},
						Verbs:     []string{"get", "patch", "update"},
					},
					{
						APIGroups: []string{"kubelb.k8c.io"},
						Resources: []string{"syncsecrets"},
						Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
					},
					{
						APIGroups: []string{"networking.k8s.io"},
						Resources: []string{"ingresses"},
						Verbs:     []string{"create", "get", "list", "watch", "patch", "update", "delete"},
					},
					{
						APIGroups: []string{"networking.k8s.io"},
						Resources: []string{"ingresses/status"},
						Verbs:     []string{"get", "patch", "update"},
					},
				}

				if dc.Spec.KubeLB != nil && dc.Spec.KubeLB.EnableSecretSynchronizer {
					r.Rules = append(r.Rules, rbacv1.PolicyRule{
						APIGroups: []string{""},
						Resources: []string{"secrets"},
						Verbs:     []string{"get", "list", "watch", "create", "update", "delete", "patch"},
					})
				}

				if cluster.Spec.KubeLB != nil && cluster.Spec.KubeLB.EnableGatewayAPI != nil && *cluster.Spec.KubeLB.EnableGatewayAPI {
					r.Rules = append(r.Rules, rbacv1.PolicyRule{
						APIGroups: []string{"gateway.networking.k8s.io"},
						Resources: []string{"gateways", "grpcroutes", "httproutes", "tcproutes", "udproutes", "tlsroutes"},
						Verbs:     []string{"get", "list", "watch", "create", "update", "delete", "patch"},
					})
					r.Rules = append(r.Rules, rbacv1.PolicyRule{
						APIGroups: []string{"gateway.networking.k8s.io"},
						Resources: []string{"gateways/status", "grpcroutes/status", "httproutes/status", "tcproutes/status", "udproutes/status", "tlsroutes/status"},
						Verbs:     []string{"get", "patch", "update"},
					})
					r.Rules = append(r.Rules, rbacv1.PolicyRule{
						APIGroups: []string{"apiextensions.k8s.io"},
						Resources: []string{"customresourcedefinitions"},
						Verbs:     []string{"get", "list", "watch", "create", "update"},
					})
				}
				return r, nil
			}
	}
}
