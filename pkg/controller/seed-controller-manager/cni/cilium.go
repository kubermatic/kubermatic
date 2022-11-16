package cni

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
)

func getCiliumOverrideValues(cluster *kubermaticv1.Cluster, overwriteRegistry string) map[string]interface{} {
	values := map[string]interface{}{
		"cni": map[string]interface{}{
			// we run Cilium as non-exclusive CNI to allow for Multus use-cases
			"exclusive": "false",
		},
		"operator": map[string]interface{}{
			"replicas": "1",
			"securityContext": map[string]interface{}{
				"seccompProfile": map[string]interface{}{
					"type": "RuntimeDefault",
				},
			},
		},
		// TODO (rastislavs): Move Hubble config (+ operator replicas) to ApplicationDefinition defaultValues instead
		"hubble": map[string]interface{}{
			"relay": map[string]interface{}{
				"enabled": "true",
			},
			"ui": map[string]interface{}{
				"enabled": "true",
			},
			// cronJob TLS cert gen method needs to be used for backward compatibility with older KKP
			"tls": map[string]interface{}{
				"auto": map[string]interface{}{
					"method": "cronJob",
				},
			},
		},
	}

	if cluster.Spec.ClusterNetwork.ProxyMode == resources.EBPFProxyMode {
		values["kubeProxyReplacement"] = "strict"
		values["k8sServiceHost"] = cluster.Status.Address.ExternalName
		values["k8sServicePort"] = fmt.Sprintf("%d", cluster.Status.Address.Port)
	} else {
		values["kubeProxyReplacement"] = "disabled"
	}

	ipamOperator := map[string]interface{}{
		"clusterPoolIPv4PodCIDR":  cluster.Spec.ClusterNetwork.Pods.GetIPv4CIDR(),
		"clusterPoolIPv4MaskSize": fmt.Sprintf("%d", *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4),
	}
	if cluster.IsDualStack() {
		values["ipv6"] = map[string]interface{}{"enabled": "true"}
		ipamOperator["clusterPoolIPv6PodCIDR"] = cluster.Spec.ClusterNetwork.Pods.GetIPv6CIDR()
		ipamOperator["clusterPoolIPv6MaskSize"] = fmt.Sprintf("%d", *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6)
	}
	values["ipam"] = map[string]interface{}{"operator": ipamOperator}

	// TODO (rastislavs): override *.image.repository + set *.image.useDigest=false if registry override is configured

	return values
}
