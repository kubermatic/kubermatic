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

package usersshkeys

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NetworkPolicyCreator NetworkPolicy allows egress traffic of user ssh keys agent to the world
func NetworkPolicyCreator(k8sApiIP string, k8sApiPort int, k8sServiceApi string) reconciling.NamedNetworkPolicyCreatorGetter {
	return func() (string, reconciling.NetworkPolicyCreator) {
		apiServicePort := intstr.FromInt(443)
		apiPort := intstr.FromInt(k8sApiPort)
		protoTcp := v1.ProtocolTCP

		return "user-ssh-key-agent", func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {

			np.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{resources.AppLabelKey: "user-ssh-keys-agent"},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
					networkingv1.PolicyTypeIngress,
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{},
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
								Port:     &apiPort,
								Protocol: &protoTcp,
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
								Port:     &apiServicePort,
								Protocol: &protoTcp,
							},
						},
					},
				},
			}
			return np, nil
		}
	}
}
