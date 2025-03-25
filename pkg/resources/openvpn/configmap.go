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

package openvpn

import (
	"fmt"
	"net"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

type serverClientConfigsData interface {
	Cluster() *kubermaticv1.Cluster
	NodeAccessNetwork() string
}

// ServerClientConfigsConfigMapReconciler returns a ConfigMap containing the ClientConfig for the OpenVPN server. It lives inside the seed-cluster.
func ServerClientConfigsConfigMapReconciler(data serverClientConfigsData) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return resources.OpenVPNClientConfigsConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			cm.Labels = resources.BaseAppLabels(name, nil)

			var iroutes []string

			// iroute for pod network
			if len(data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks) < 1 {
				return nil, fmt.Errorf("cluster.Spec.ClusterNetwork.Pods.CIDRBlocks must contain at least one entry")
			}
			_, podNet, err := net.ParseCIDR(data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0])
			if err != nil {
				return nil, err
			}
			iroutes = append(iroutes, fmt.Sprintf("iroute %s %s",
				podNet.IP.String(),
				net.IP(podNet.Mask).String()))

			// iroute for service network
			if len(data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks) < 1 {
				return nil, fmt.Errorf("cluster.Spec.ClusterNetwork.Services.CIDRBlocks must contain at least one entry")
			}
			_, serviceNet, err := net.ParseCIDR(data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks[0])
			if err != nil {
				return nil, err
			}
			iroutes = append(iroutes, fmt.Sprintf("iroute %s %s",
				serviceNet.IP.String(),
				net.IP(serviceNet.Mask).String()))

			_, nodeAccessNetwork, err := net.ParseCIDR(data.NodeAccessNetwork())
			if err != nil {
				return nil, fmt.Errorf("failed to parse node access network %s: %w", data.NodeAccessNetwork(), err)
			}
			iroutes = append(iroutes, fmt.Sprintf("iroute %s %s",
				nodeAccessNetwork.IP.String(),
				net.IP(nodeAccessNetwork.Mask).String()))

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			// trailing newline
			iroutes = append(iroutes, "")
			cm.Data["user-cluster-client"] = strings.Join(iroutes, "\n")

			return cm, nil
		}
	}
}
