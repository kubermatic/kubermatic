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
	"strings"
	"text/template"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

var envoyConfigTemplate = `admin:
  access_log_path: /dev/stdout
  address:
    socket_address:
      protocol: TCP
      address: 127.0.0.1
      port_value: {{.AdminPort}}
static_resources:
  listeners: {{if not .Listeners -}}[]{{- end}}
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
            http2_protocol_options: {}
      load_assignment:
        cluster_name: proxy_cluster
        endpoints:
          - lb_endpoints:
              - endpoint:
                  address:
                    socket_address:
                      address: {{.ProxyHost}}
                      port_value: {{.ProxyPort}}
`

type Config struct {
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

// ConfigMapCreator returns a ConfigMap containing the config for the Envoy agent
func ConfigMapCreator(cfg Config) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.EnvoyAgentConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			cm.Labels = resources.BaseAppLabels(resources.EnvoyAgentConfigMapName, nil)
			var b strings.Builder
			err := template.Must(template.New("envoy-config").Parse(envoyConfigTemplate)).Execute(&b, cfg)
			if err != nil {
				return nil, err
			}
			cm.Data["envoy.yaml"] = b.String()

			return cm, nil
		}
	}
}
