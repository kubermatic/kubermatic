/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package apiserver

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	stackscommon "k8c.io/kubermatic/v2/pkg/install/stack/common"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOIDCIssuerAllowReconciler(t *testing.T) {
	loadBalancerBackendPeer := networkingv1.NetworkPolicyPeer{
		NamespaceSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				corev1.LabelMetadataName: "tenant-a",
			},
		},
		PodSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				"app.kubernetes.io/name": "tenant-a",
			},
		},
	}

	policyName, reconciler := OIDCIssuerAllowReconciler(
		[]net.IP{net.ParseIP("10.10.45.0")},
		[]networkingv1.NetworkPolicyPeer{loadBalancerBackendPeer},
		"",
	)()
	require.Equal(t, resources.NetworkPolicyOIDCIssuerAllow, policyName)

	networkPolicy, err := reconciler(&networkingv1.NetworkPolicy{})
	require.NoError(t, err)

	require.Equal(t, networkingv1.NetworkPolicySpec{
		PolicyTypes: []networkingv1.PolicyType{
			networkingv1.PolicyTypeEgress,
		},
		PodSelector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				resources.AppLabelKey: name,
			},
		},
		Egress: []networkingv1.NetworkPolicyEgressRule{
			{
				To: []networkingv1.NetworkPolicyPeer{
					{
						IPBlock: &networkingv1.IPBlock{
							CIDR: "10.10.45.0/32",
						},
					},
				},
			},
			{
				To: []networkingv1.NetworkPolicyPeer{
					loadBalancerBackendPeer,
				},
			},
			{
				To: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								corev1.LabelMetadataName: stackscommon.NginxIngressControllerNamespace,
							},
						},
					},
				},
			},
			{
				To: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								corev1.LabelMetadataName: stackscommon.EnvoyGatewayControllerNamespace,
							},
						},
					},
				},
			},
		},
	}, networkPolicy.Spec)
}
