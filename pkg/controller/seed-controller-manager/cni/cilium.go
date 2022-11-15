package cni

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
)

func getCiliumOverrideValues(cluster *kubermaticv1.Cluster) map[string]interface{} {
	values := map[string]interface{}{
		"cni": map[string]interface{}{
			"exclusive": "false", // non-exclusive to allow Multus use-cases
		},
		"operator": map[string]interface{}{
			"replicas": "1",
			"securityContext": map[string]interface{}{
				"seccompProfile": map[string]interface{}{
					"type": "RuntimeDefault",
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
