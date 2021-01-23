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

package envoyagent

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// ConfigMapCreator returns a ConfigMap containing the config for the Envoy agent
func ConfigMapCreator(namespace, url string) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.EnvoyAgentConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Labels = resources.BaseAppLabels(resources.EnvoyAgentConfigMapName, nil)
			envoy_config := fmt.Sprintf(`
admin:
  access_log_path: /tmp/admin_access.log
  address:
    socket_address:
      protocol: TCP
      address: 127.0.0.1
      port_value: 9902
static_resources:
  listeners:
  - name: listener_0
    address:
      socket_address:
        protocol: TCP
        address: 0.0.0.0
        port_value: 6443
    filter_chains:
    - filters:
      - name: tcp
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
          stat_prefix: tcp_stats
          cluster: "cluster_0"
          tunneling_config:
			hostname: "apiserver.%s.svc.cluster.local:6443"
  	    - name: listener_1
    address:
      socket_address:
        protocol: TCP
        address: 0.0.0.0
        port_value: 6444
    filter_chains:
    - filters:
      - name: tcp
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
          stat_prefix: tcp_stats
          cluster: "cluster_0"
          tunneling_config:
			hostname: "openvpn-server.%s.svc.cluster.local:1194"
  clusters:
    - name: cluster_0
      connect_timeout: 5s
      type: LOGICAL_DNS
      dns_lookup_family: V4_ONLY
      http2_protocol_options:
        {}
      load_assignment:
        cluster_name: cluster_0
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: %s
                      port_value: 8080
      `, namespace, namespace, url)

			cm.Data["envoy.yaml"] = envoy_config

			return cm, nil
		}
	}
}
