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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// clusterIsolationNetworkPolicyReconciler creates a network policy that restrict Egress traffic. By default it allows access to
// each other pod in the same namespace, Internet, Kubermatic nodeport proxy and DNS servers.
func clusterIsolationNetworkPolicyReconciler(clusterIP string, nameservers []string) reconciling.NamedNetworkPolicyReconcilerFactory {
	apiServerCIDR := clusterIP + "/32"
	dnsPort := intstr.FromInt(53)
	tcp := corev1.ProtocolTCP
	udp := corev1.ProtocolUDP

	// This address might only be set after the control plane has initialized.
	var apiServerRule networkingv1.NetworkPolicyEgressRule
	if clusterIP != "" {
		apiServerRule = networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: apiServerCIDR,
					},
				},
			},
		}
	}

	var dnsRules []networkingv1.NetworkPolicyEgressRule
	for _, ns := range nameservers {
		dnsRules = append(dnsRules, networkingv1.NetworkPolicyEgressRule{
			To: []networkingv1.NetworkPolicyPeer{
				{
					IPBlock: &networkingv1.IPBlock{
						CIDR: ns + "/32",
					},
				},
			},
			Ports: []networkingv1.NetworkPolicyPort{
				{
					Protocol: &tcp,
					Port:     &dnsPort,
				},
				{
					Protocol: &udp,
					Port:     &dnsPort,
				},
			},
		})
	}

	return func() (string, reconciling.NetworkPolicyReconciler) {
		return "cluster-isolation", func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
				Egress: []networkingv1.NetworkPolicyEgressRule{
					// Allow access to pod's in same namespace
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								PodSelector: &metav1.LabelSelector{},
							},
						},
					},
					// Allow public ip addresses
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								IPBlock: &networkingv1.IPBlock{
									CIDR:   "0.0.0.0/0",
									Except: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"},
								},
							},
						},
					},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
				},
			}

			// Allow kubermatic nodeport-proxy ip address
			np.Spec.Egress = append(np.Spec.Egress, apiServerRule)

			// Allow dns servers
			np.Spec.Egress = append(np.Spec.Egress, dnsRules...)
			return np, nil
		}
	}
}

// clusterImporterNetworkPolicyReconciler creates a special network policy that allows the importer pod to access
// the image-repo in the cluster.
func clusterImporterNetworkPolicyReconciler() reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return "allow-kubevirt-importer", func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Spec = networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":             "containerized-data-importer",
						"cdi.kubevirt.io": "importer",
					},
				},
				Egress: []networkingv1.NetworkPolicyEgressRule{
					// Allow access to pod in any namespace with specific label
					{
						To: []networkingv1.NetworkPolicyPeer{
							{
								NamespaceSelector: &metav1.LabelSelector{},
								PodSelector: &metav1.LabelSelector{
									MatchLabels: map[string]string{
										"app": "image-repo",
									},
								},
							},
						},
					},
				},
				PolicyTypes: []networkingv1.PolicyType{
					networkingv1.PolicyTypeEgress,
				},
			}

			return np, nil
		}
	}
}

func customNetworkPolicyReconciler(existing kubermaticv1.CustomNetworkPolicy) reconciling.NamedNetworkPolicyReconcilerFactory {
	return func() (string, reconciling.NetworkPolicyReconciler) {
		return existing.Name, func(np *networkingv1.NetworkPolicy) (*networkingv1.NetworkPolicy, error) {
			np.Name = existing.Name
			np.Spec = existing.Spec
			return np, nil
		}
	}
}

func reconcileClusterIsolationNetworkPolicy(ctx context.Context, cluster *kubermaticv1.Cluster, dc *kubermaticv1.DatacenterSpecKubevirt, client ctrlruntimeclient.Client, namespace string) error {
	var nameservers []string
	if dc.DNSConfig != nil {
		nameservers = dc.DNSConfig.Nameservers
	}

	namedNetworkPolicyReconcilerFactories := []reconciling.NamedNetworkPolicyReconcilerFactory{
		clusterIsolationNetworkPolicyReconciler(cluster.Status.Address.IP, nameservers),
		clusterImporterNetworkPolicyReconciler(),
	}
	if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyReconcilerFactories, namespace, client); err != nil {
		return fmt.Errorf("failed to ensure Network Policies: %w", err)
	}
	return nil
}

func reconcileCustomNetworkPolicies(ctx context.Context, cluster *kubermaticv1.Cluster, dc *kubermaticv1.DatacenterSpecKubevirt, client ctrlruntimeclient.Client, namespace string) error {
	namedNetworkPolicyReconcilerFactories := make([]reconciling.NamedNetworkPolicyReconcilerFactory, 0)
	for _, netpol := range dc.CustomNetworkPolicies {
		namedNetworkPolicyReconcilerFactories = append(namedNetworkPolicyReconcilerFactories, customNetworkPolicyReconciler(netpol))
	}
	if err := reconciling.ReconcileNetworkPolicies(ctx, namedNetworkPolicyReconcilerFactories, namespace, client); err != nil {
		return fmt.Errorf("failed to ensure Custom Network Policies: %w", err)
	}
	return nil
}
