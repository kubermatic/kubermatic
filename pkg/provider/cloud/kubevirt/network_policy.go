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
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func clusterIsolationNetworkPolicyReconciler() reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return "cluster-isolation", func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{
						From: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: &metav1.LabelSelector{},
							},
						},
					},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeIngress,
				},
			}
			return np, nil
		}
	}
}

func customNetworkPolicyReconciler(existing *kubermaticv1.CustomNetworkPolicy) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return existing.Name, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Name = existing.Name
			np.Spec = existing.Spec
			return np, nil
		}
	}
}

func reconcileClusterIsolationNetworkPolicy(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) error {
	namedNetworkPolicyReconcilerFactorys := []reconciling.NamedNetworkPolicyReconcilerFactory{
		clusterIsolationNetworkPolicyReconciler(),
	}
	if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyReconcilerFactorys, cluster.Status.NamespaceName, client); err != nil {
		return fmt.Errorf("failed to ensure Network Policies: %w", err)
	}
	return nil
}

func reconcileCustomNetworkPolicies(ctx context.Context, cluster *kubermaticv1.Cluster, dc *kubermaticv1.DatacenterSpecKubevirt, client ctrlruntimeclient.Client) error {
	namedNetworkPolicyReconcilerFactorys := make([]reconciling.NamedNetworkPolicyReconcilerFactory, 0)
	for _, netpol := range dc.CustomNetworkPolicies {
		namedNetworkPolicyReconcilerFactorys = append(namedNetworkPolicyReconcilerFactorys, customNetworkPolicyReconciler(netpol))
	}
	if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyReconcilerFactorys, cluster.Status.NamespaceName, client); err != nil {
		return fmt.Errorf("failed to ensure Custom Network Policies: %w", err)
	}
	return nil
}
