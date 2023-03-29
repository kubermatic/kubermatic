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

	"k8c.io/kubermatic/v3/pkg/controller/operator/common"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// KubeDNSNetworkPolicyReconciler NetworkPolicy allows ingress traffic to coredns on port 53 TCP/UDP and egress to anywhere on port 53 TCP/UDP.
func KubeDNSNetworkPolicyReconciler(k8sApiIP string, k8sApiPort int, k8sServiceApi string) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		dnsPort := intstr.FromInt(53)
		apiServicePort := intstr.FromInt(443)
		apiPort := intstr.FromInt(k8sApiPort)
		metricsPort := intstr.FromInt(9153)
		protoUdp := corev1.ProtocolUDP
		protoTcp := corev1.ProtocolTCP

		return "kube-dns", func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
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
					{
						From: []networkingv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{common.ComponentLabel: resources.MLAComponentName},
								},
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{common.ComponentLabel: resources.MLAComponentName},
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: &protoTcp,
								Port:     &metricsPort,
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
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								IPBlock: &networkingv1.IPBlock{
									CIDR: fmt.Sprintf("%s/32", k8sServiceApi),
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: &protoTcp,
								Port:     &apiServicePort,
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
