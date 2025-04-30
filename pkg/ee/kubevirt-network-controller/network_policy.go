//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

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

package kubevirtnetworkcontroller

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// namespacedClusterIsolationNetworkPolicyReconciler creates a network policy that restrict Egress traffic between clusters deployed in the namespaced mode within the same VPC network.
func namespacedClusterIsolationNetworkPolicyReconciler(clusterName string, subnets []string, subnetGateways []string) reconciling.NamedNetworkPolicyReconcilerFactory {
	// Allow egress for subnet gateways
	subnetGatewaysRule := []networkingv1.NetworkPolicyEgressRule{}
	for _, gw := range subnetGateways {
		if gw != "" {
			subnetGatewaysRule = append(subnetGatewaysRule, networkingv1.NetworkPolicyEgressRule{
				To: []networkingv1.NetworkPolicyPeer{
					{
						IPBlock: &networkingv1.IPBlock{
							CIDR: gw + "/32",
						},
					},
				},
			})
		}
	}

	// Allow egress for anything but other workload subnets in the same vpc.
	subnetRule := networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{
			{
				IPBlock: &networkingv1.IPBlock{
					CIDR:   "0.0.0.0/0",
					Except: subnets,
				},
			},
		},
	}

	return func() (string, reconciling.NetworkPolicyReconciler) {
		return fmt.Sprintf("cluster-isolation-%s", clusterName), func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"cluster.x-k8s.io/cluster-name": clusterName,
					},
				},
				Egress: []networkingv1.NetworkPolicyEgressRule{
					// Allow egress for pods with the cluster.x-k8s.io/cluster-name label
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"cluster.x-k8s.io/cluster-name": clusterName,
									},
								},
							},
						},
					},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
				},
			}
			np.Spec.Egress = append(np.Spec.Egress, subnetGatewaysRule...)
			np.Spec.Egress = append(np.Spec.Egress, subnetRule)
			return np, nil
		}
	}
}

func reconcileNamespacedClusterIsolationNetworkPolicy(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, subnets []string, subnetGateways []string, namespace string) error {
	namedNetworkPolicyReconcilerFactories := []reconciling.NamedNetworkPolicyReconcilerFactory{
		namespacedClusterIsolationNetworkPolicyReconciler(cluster.Name, subnets, subnetGateways),
	}
	if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyReconcilerFactories, namespace, client); err != nil {
		return fmt.Errorf("failed to ensure Network Policies: %w", err)
	}
	return nil
}
