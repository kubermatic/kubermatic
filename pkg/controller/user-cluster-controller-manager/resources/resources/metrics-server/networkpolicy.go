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

package metricsserver

import (
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// NetworkPolicyReconciler NetworkPolicy allows egress traffic of user ssh keys agent to the world.
func NetworkPolicyReconciler() reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return "metrics-server", func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			metricsPort := intstr.FromInt(9153)
			httpsPort := intstr.FromInt(servingPort)
			protoTCP := corev1.ProtocolTCP

			np.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{resources.AppLabelKey: resources.MetricsServerDeploymentName},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
					networkingv1.PolicyTypeIngress,
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{
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
								Port:     &metricsPort,
								Protocol: &protoTCP,
							},
						},
					},
					{
						From: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{resources.AppLabelKey: resources.KonnectivityDeploymentName},
								},
							},
						},
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Port:     &httpsPort,
								Protocol: &protoTCP,
							},
						},
					},
				},
				Egress: []networkingv1.NetworkPolicyEgressRule{{}},
			}
			return np, nil
		}
	}
}
