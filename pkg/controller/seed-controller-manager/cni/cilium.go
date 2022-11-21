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
