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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	kubermaticmaster "k8c.io/kubermatic/v2/pkg/install/stack/kubermatic-master"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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
										"cluster":             c.ObjectMeta.Name,
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
			protoUdp := corev1.ProtocolUDP
			protoTcp := corev1.ProtocolTCP

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
								Protocol: &protoUdp,
								Port:     &dnsPort,
							},
							{
								Protocol: &protoTcp,
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
										"cluster":             c.ObjectMeta.Name,
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

func MachineControllerWebhookReconciler(c *kubermaticv1.Cluster) reconciling.NamedNetworkPolicyReconcilerFactory {
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

func UserClusterWebhookReconciler(c *kubermaticv1.Cluster) reconciling.NamedNetworkPolicyReconcilerFactory {
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

// ClusterExternalAddrAllowReconciler returns a func to create/update the apiserver cluster-external-addr-allow egress policy.
// This policy is necessary in Konnectivity setup, so that konnectivity-server can connect to the apiserver via
// the external URL (used as service-account-issuer) to validate konnectivity-agent authentication token.
func ClusterExternalAddrAllowReconciler(egressIPs []net.IP, exposeStrategy kubermaticv1.ExposeStrategy) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return resources.NetworkPolicyClusterExternalAddrAllow, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
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
						To: ipListToPeers(egressIPs),
					},
				},
			}

			// allow egress traffic to the nodeport-proxy as for some CNI + kube-proxy mode
			// combinations a local path to it may be used to reach the external apiserver address
			if exposeStrategy == kubermaticv1.ExposeStrategyLoadBalancer {
				// allows traffic to the nodeport-proxy running in the user cluster namespace used for the LB expose strategy
				np.Spec.Egress[0].To = append(np.Spec.Egress[0].To, networkingv1.NetworkPolicyPeer{
					PodSelector: &metav1.LabelSelector{
						MatchLabels: resources.BaseAppLabels(resources.NodePortProxyEnvoyDeploymentName, nil),
					},
				})
			} else {
				// allows traffic to the nodeport-proxy running in the kubermatic namespace used for other expose strategies
				np.Spec.Egress[0].To = append(np.Spec.Egress[0].To, networkingv1.NetworkPolicyPeer{
					NamespaceSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							corev1.LabelMetadataName: resources.KubermaticNamespace,
						},
					},
					PodSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							common.NameLabel: nodeportproxy.EnvoyDeploymentName,
						},
					},
				})
			}

			return np, nil
		}
	}
}

// OIDCIssuerAllowReconciler returns a func to create/update the apiserver oidc-issuer-allow egress policy.
func OIDCIssuerAllowReconciler(egressIPs []net.IP) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
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
							// allow egress traffic to the nginx-ingress-controller as for some CNI + kube-proxy
							// mode combinations a local path to it may be used to reach OIDC issuer installed in KKP
							NamespaceSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									corev1.LabelMetadataName: kubermaticmaster.NginxIngressControllerNamespace,
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
