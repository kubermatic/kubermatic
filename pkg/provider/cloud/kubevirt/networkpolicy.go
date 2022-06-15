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
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func clusterIsolationNetworkPolicyCreator() reconciling.NamedNetworkPolicyCreatorGetter {
	return func() (string, reconciling.NetworkPolicyCreator) {
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
			}
			return np, nil
		}
	}
}

func reconcileNetworkPolicy(ctx context.Context, cluster *kubermaticv1.Cluster, client ctrlruntimeclient.Client) error {
	namedNetworkPolicyCreatorGetters := []reconciling.NamedNetworkPolicyCreatorGetter{
		clusterIsolationNetworkPolicyCreator(),
	}
	if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyCreatorGetters, cluster.Status.NamespaceName, client); err != nil {
		return fmt.Errorf("failed to ensure Network Policies: %w", err)
	}
	return nil
}
