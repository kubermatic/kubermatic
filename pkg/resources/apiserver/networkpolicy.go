/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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
	"fmt"
	"net"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// DenyAllPolicyReconciler returns a func to create/update the apiserver
// deny all egress policy.
func DenyAllPolicyReconciler() reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyDefaultDenyAllEgress, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
				},
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						resources.AppLabelKey: name,
					},
				},
				Egress: []networkingv1.NetworkPolicyEgressRule{},
			}

			return np, nil
		}
	}
}

// EctdAllowReconciler returns a func to create/update the apiserver ETCD allow egress policy.
func EctdAllowReconciler(c *kubermaticv1.Cluster) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyEtcdAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
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
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										resources.AppLabelKey: "etcd",
										"cluster":             c.Name,
									},
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

// DNSAllowReconciler returns a func to create/update the apiserver DNS allow egress policy.
func DNSAllowReconciler(c *kubermaticv1.Cluster, data *resources.TemplateData) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyDNSAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			dnsPort := intstr.FromInt(53)

			np.Spec = networkingv1.NetworkPolicySpec{
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
						Ports: []networkingv1.NetworkPolicyPort{
							{
								Protocol: ptr.To(corev1.ProtocolUDP),
								Port:     &dnsPort,
							},
							{
								Protocol: ptr.To(corev1.ProtocolTCP),
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

// OpenVPNServerAllowReconciler returns a func to create/update the apiserver OpenVPN allow egress policy.
func OpenVPNServerAllowReconciler(c *kubermaticv1.Cluster) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyOpenVPNServerAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
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
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										resources.AppLabelKey: "openvpn-server",
										"cluster":             c.Name,
									},
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

func MachineControllerWebhookAllowReconciler(c *kubermaticv1.Cluster) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyMachineControllerWebhookAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
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
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										resources.AppLabelKey: "machine-controller-webhook",
									},
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

func UserClusterWebhookAllowReconciler(c *kubermaticv1.Cluster) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyUserClusterWebhookAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
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
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										resources.AppLabelKey: resources.UserClusterWebhookDeploymentName,
									},
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

func OSMWebhookAllowReconciler(c *kubermaticv1.Cluster) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyOperatingSystemManagerWebhookAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
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
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										resources.AppLabelKey: resources.OperatingSystemManagerWebhookDeploymentName,
									},
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

func MetricsServerAllowReconciler(c *kubermaticv1.Cluster) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyMetricsServerAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
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
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										resources.AppLabelKey: "metrics-server",
									},
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

// ApiserverInternalAllowReconciler returns a func to create/update the apiserver-internal-allow egress policy.
// This policy is necessary since konnectivity-server (sidecar to kube-apiserver when konnectivity is enabled) needs
// to talk to the Kubernetes API to validate tokens coming from konnectivity-agent.
//
// This was previously handled with a policy called cluster-external-addr-allow that allowed connection to the
// the external endpoint, but no reasoning for this design choice could be found in code comments or PR descriptions.
// Upstream itself uses localhost in an example (see https://github.com/kubernetes-sigs/apiserver-network-proxy/blob/a38752dc9884a1fc1c32652eacb38aed21e4ab25/examples/kubernetes/kubeconfig#L11),
// so the strong assumption here is that this was never necessary.
func ApiserverInternalAllowReconciler() reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyApiserverInternalAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
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
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										resources.AppLabelKey: name,
									},
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

// OIDCIssuerAllowReconciler returns a func to create/update the apiserver oidc-issuer-allow egress policy.
func OIDCIssuerAllowReconciler(egressIPs []net.IP, namespaceOverride string) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		ingressNamespace := kubermaticmaster.NginxIngressControllerNamespace
		if namespaceOverride != "" {
			ingressNamespace = namespaceOverride
		}

		return resources.NetworkPolicyOIDCIssuerAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
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
						To: append(ipListToPeers(egressIPs), networkingv1.NetworkPolicyPeer{
							// allow egress traffic to the ingress-controller as for some CNI + kube-proxy
							// mode combinations a local path to it may be used to reach OIDC issuer installed in KKP
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									corev1.LabelMetadataName: ingressNamespace,
								},
							},
						}),
					},
				},
			}

			return np, nil
		}
	}
}

func SeedApiserverAllowReconciler(endpoints []net.IP) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicySeedApiserverAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
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
						To: ipListToPeers(endpoints),
					},
				},
			}

			return np, nil
		}
	}
}

func ipListToPeers(ips []net.IP) []networkingv1.NetworkPolicyPeer {
	result := []networkingv1.NetworkPolicyPeer{}

	for _, ip := range ips {
		cidr := fmt.Sprintf("%s/%d", ip.String(), net.IPv4len*8)
		if ip.To4() == nil {
			cidr = fmt.Sprintf("%s/%d", ip.String(), net.IPv6len*8)
		}
		result = append(result, networkingv1.NetworkPolicyPeer{
			IPBlock: &networkingv1.IPBlock{
				CIDR: cidr,
			},
		})
	}

	return result
}
