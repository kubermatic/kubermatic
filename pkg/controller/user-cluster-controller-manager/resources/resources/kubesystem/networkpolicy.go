/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package kubesystem

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	nodelocaldns "k8c.io/kubermatic/v2/pkg/controller/user-cluster-controller-manager/resources/resources/node-local-dns"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// DefaultNetworkPolicyCreator Default policy creator denys all expect egress to kube-dns for all pods without any network policy applied.
func DefaultNetworkPolicyCreator() reconciling.NamedNetworkPolicyCreatorGetter {
	return func() (string, reconciling.NetworkPolicyCreator) {
		return "allow-dns", func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			dnsPort := intstr.FromInt(53)
			protoUdp := v1.ProtocolUDP
			protoTcp := v1.ProtocolTCP

			// dns access to node local dns cache
			np.Spec = networkingv1.NetworkPolicySpec{
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeIngress,
					networkingv1.PolicyTypeEgress,
				},
				PodSelector: metav1.LabelSelector{},
				Ingress:     []networkingv1.NetworkPolicyIngressRule{},
				Egress: []networkingv1.NetworkPolicyEgressRule{
					{
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: &protoTcp,
								Port:     &dnsPort,
							},
							{
								Protocol: &protoUdp,
								Port:     &dnsPort,
							},
						},
						To: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{common.NameLabel: "kube-dns"},
								},
							},
						},
					},
					{
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: &protoTcp,
								Port:     &dnsPort,
							},
							{
								Protocol: &protoUdp,
								Port:     &dnsPort,
							},
						},
						To: []networkingv1.NetworkPolicyPeer{
							{
								IPBlock: &networkingv1.IPBlock{
									CIDR: fmt.Sprintf("%s/32", nodelocaldns.CacheAddress),
								},
							},
						},
					},
				},
			}

			return np, nil
		}
	}
}
