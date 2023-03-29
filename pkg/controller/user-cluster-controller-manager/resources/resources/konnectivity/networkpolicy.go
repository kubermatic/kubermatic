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

package konnectivity

import (
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkPolicyReconciler NetworkPolicy allows all egress traffic.
func NetworkPolicyReconciler() reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return "konnectivity", func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{resources.AppLabelKey: "konnectivity-agent"},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
					networkingv1.PolicyTypeIngress,
				},
				Ingress: []networkingv1.NetworkPolicyIngressRule{},
				Egress:  []networkingv1.NetworkPolicyEgressRule{{}},
			}
			return np, nil
		}
	}
}
