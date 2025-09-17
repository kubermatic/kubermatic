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
	"strings"
	"text/template"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const StatsPort uint32 = 8002

var envoyConfigTemplate = `admin:
  access_log_path: /dev/stdout
  address:
    socket_address:
      protocol: TCP
      address: 127.0.0.1
      port_value: {{.AdminPort}}
static_resources:
  listeners:
  - name: service_stats
    address:
      socket_address:
        protocol: TCP
        address: 0.0.0.0
        port_value: {{.StatsPort}}
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: service_stats
          route_config:
            name: local_route
            virtual_hosts:
            - name: stats_backend
              domains: ["*"]
              routes:
              - match:
                  prefix: "/stats"
                route:
                  cluster: service_stats
          http_filters:
          - name: envoy.filters.http.router
            "typed_config":
              "@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router"
{{- range $i, $l := .Listeners}}
  - name: listener_{{$i}}
    address:
      socket_address:
        protocol: TCP
        address: {{$l.BindAddress}}
        port_value: {{$l.BindPort}}
    filter_chains:
    - filters:
      - name: tcp
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.tcp_proxy.v3.TcpProxy
          stat_prefix: tcp_stats
          cluster: "proxy_cluster"
          tunneling_config:
            hostname: {{$l.Authority}}
{{- end}}
  clusters:
    - name: proxy_cluster
      connect_timeout: 5s
      type: LOGICAL_DNS
      # This ensures HTTP/2 CONNECT is used for establishing the tunnel.
      typed_extension_protocol_options:
        envoy.extensions.upstreams.http.v3.HttpProtocolOptions:
          "@type": type.googleapis.com/envoy.extensions.upstreams.http.v3.HttpProtocolOptions
          explicit_http_config:
            http2_protocol_options:
              connection_keepalive:
                interval: 5s
                timeout: 2s
      load_assignment:
        cluster_name: proxy_cluster
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: {{.ProxyHost}}
                      port_value: {{.ProxyPort}}
    - name: service_stats
      connect_timeout: 0.1s
      type: STATIC
      load_assignment:
        cluster_name: service_stats
        endpoints:
        - lb_endpoints:
            - endpoint:
                address:
                  socket_address:
                    address: 127.0.0.1
                    port_value: {{.AdminPort}}
`

type Config struct {
	StatsPort uint32
	AdminPort uint32
	ProxyHost string
	ProxyPort uint32
	Listeners []Listener
}

type Listener struct {
	BindAddress string
	BindPort    uint32
	Authority   string
}

// ConfigMapReconciler returns a ConfigMap containing the config for the Envoy agent.
func ConfigMapReconciler(cfg Config) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return resources.EnvoyAgentConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			// ensure StatsPort is not used by any listener
			for _, listener := range cfg.Listeners {
				if listener.BindPort == StatsPort {
					return nil, fmt.Errorf("listener port \"%d\" is reserved and must not be used", listener.BindPort)
				}
			}
			cfg.StatsPort = StatsPort

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Labels = resources.BaseAppLabels(resources.EnvoyAgentConfigMapName, nil)
			var b strings.Builder
			err := template.Must(template.New("envoy-config").Parse(envoyConfigTemplate)).Execute(&b, cfg)
			if err != nil {
				return nil, err
			}
			cm.Data[resources.EnvoyAgentConfigFileName] = b.String()

			return cm, nil
		}
	}
}
