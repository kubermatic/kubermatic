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

package coredns

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
)

// AllowAllDnsNetworkPolicyCreator allows egress traffic from all pods to kube-dns
func AllowAllDnsNetworkPolicyCreator() reconciling.NamedNetworkPolicyCreatorGetter {
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
				Ingress: []networkingv1.NetworkPolicyIngressRule{},
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
							{
								IPBlock: &networkingv1.IPBlock{
									CIDR: "169.254.20.10/32",
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

// KubeDNSNetworkPolicyCreator NetworkPolicy allows ingress traffic to coredns on port 53 TCP/UDP and egress to anywhere on port 53 TCP/UDP.
func KubeDNSNetworkPolicyCreator(k8sApiIP string, k8sApiPort int) reconciling.NamedNetworkPolicyCreatorGetter {
	return func() (string, reconciling.NetworkPolicyCreator) {
		return "kube-dns", func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			dnsPort := intstr.FromInt(53)
			apiPort := intstr.FromInt(k8sApiPort)
			protoUdp := v1.ProtocolUDP
			protoTcp := v1.ProtocolTCP

			np.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{common.NameLabel: "kube-dns"},
				},

				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
					networkingv1.PolicyTypeIngress,
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
					{
						From: []networkingv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{},
							},
						},
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
					},
					{
						From: []networkingv1.NetworkPolicyPeer{
							{
								IPBlock: &networkingv1.IPBlock{
									CIDR: "0.0.0.0/0",
								},
							},
						},
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
					},
				},
				// api access
				Egress: []networkingv1.NetworkPolicyEgressRule{
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								IPBlock: &networkingv1.IPBlock{
									CIDR: fmt.Sprintf("%s/32", k8sApiIP),
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: &protoTcp,
								Port:     &apiPort,
							},
						},
					},
					// world dns access
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								IPBlock: &networkingv1.IPBlock{
									CIDR: "0.0.0.0/0",
								},
							},
						},
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
					},
				},
			}
			return np, nil
		}
	}
}
