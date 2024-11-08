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

package kubevirtnetworkcontroller

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// namespacedClusterIsolationNetworkPolicyReconciler creates a network policy that restrict Egress traffic between clusters deployed in the namespaced mode within the same VPC network
func namespacedClusterIsolationNetworkPolicyReconciler(clusterName string, subnets []string, subnetGateways []string) reconciling.NamedNetworkPolicyReconcilerFactory {
	// Allow egress for subnet gateways
	subnetGatewaysRule := []networkingv1.NetworkPolicyEgressRule{}
	for _, gw := range subnetGateways {
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
					// Allow egress for pods with the cluster.x-k8s.io/cluster-name label and api-router pod
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
