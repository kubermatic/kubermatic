/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package prometheus

import (
	"bytes"
	"text/template"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

type Config struct {
	MLAGatewayURL       string
	TLSCertFile         string
	TLSKeyFile          string
	TLSCACertFile       string
	CustomScrapeConfigs string
}

func ConfigMapCreator(config Config) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.UserClusterPrometheusConfigMapName, func(configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if configMap.Data == nil {
				configMap.Data = map[string]string{}
			}
			t, err := template.New("prometheus").Parse(configTemplate)
			if err != nil {
				return nil, err
			}
			configBuf := bytes.Buffer{}
			if err := t.Execute(&configBuf, config); err != nil {
				return nil, err
			}
			configMap.Data["prometheus.yaml"] = configBuf.String()
			configMap.Labels = resources.BaseAppLabels(appName, nil)
			return configMap, nil
		}
	}
}

const (
	configTemplate = `
global:
  evaluation_interval: 30s
  scrape_interval: 30s
  scrape_timeout: 10s
remote_write:
- url: {{ .MLAGatewayURL }}
  tls_config:
    cert_file: {{ .TLSCertFile }}
    key_file: {{ .TLSKeyFile }}
    ca_file: {{ .TLSCACertFile }}
rule_files:
- /etc/config/recording_rules.yml
- /etc/config/alerting_rules.yml
- /etc/config/rules
- /etc/config/alerts
scrape_configs:
- job_name: prometheus
  static_configs:
  - targets:
    - localhost:9090
- bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  job_name: kubernetes-nodes
  kubernetes_sd_configs:
  - role: node
  relabel_configs:
  - action: labelmap
    regex: __meta_kubernetes_node_label_(.+)
  - replacement: kubernetes.default.svc:443
    target_label: __address__
  - regex: (.+)
    replacement: /api/v1/nodes/$1/proxy/metrics
    source_labels:
    - __meta_kubernetes_node_name
    target_label: __metrics_path__
  scheme: https
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
- bearer_token_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  job_name: kubernetes-nodes-cadvisor
  kubernetes_sd_configs:
  - role: node
  relabel_configs:
  - action: labelmap
    regex: __meta_kubernetes_node_label_(.+)
  - replacement: kubernetes.default.svc:443
    target_label: __address__
  - regex: (.+)
    replacement: /api/v1/nodes/$1/proxy/metrics/cadvisor
    source_labels:
    - __meta_kubernetes_node_name
    target_label: __metrics_path__
  scheme: https
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
- job_name: kubernetes-service-endpoints
  kubernetes_sd_configs:
  - role: endpoints
  relabel_configs:
  - action: keep
    regex: true
    source_labels:
    - __meta_kubernetes_service_annotation_prometheus_io_scrape
  - action: replace
    regex: (https?)
    source_labels:
    - __meta_kubernetes_service_annotation_prometheus_io_scheme
    target_label: __scheme__
  - action: replace
    regex: (.+)
    source_labels:
    - __meta_kubernetes_service_annotation_prometheus_io_path
    target_label: __metrics_path__
  - action: replace
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: $1:$2
    source_labels:
    - __address__
    - __meta_kubernetes_service_annotation_prometheus_io_port
    target_label: __address__
  - action: labelmap
    regex: __meta_kubernetes_service_label_(.+)
  - action: replace
    source_labels:
    - __meta_kubernetes_namespace
    target_label: kubernetes_namespace
  - action: replace
    source_labels:
    - __meta_kubernetes_service_name
    target_label: kubernetes_name
  - action: replace
    source_labels:
    - __meta_kubernetes_pod_node_name
    target_label: kubernetes_node
- job_name: kubernetes-service-endpoints-slow
  kubernetes_sd_configs:
  - role: endpoints
  relabel_configs:
  - action: keep
    regex: true
    source_labels:
    - __meta_kubernetes_service_annotation_prometheus_io_scrape_slow
  - action: replace
    regex: (https?)
    source_labels:
    - __meta_kubernetes_service_annotation_prometheus_io_scheme
    target_label: __scheme__
  - action: replace
    regex: (.+)
    source_labels:
    - __meta_kubernetes_service_annotation_prometheus_io_path
    target_label: __metrics_path__
  - action: replace
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: $1:$2
    source_labels:
    - __address__
    - __meta_kubernetes_service_annotation_prometheus_io_port
    target_label: __address__
  - action: labelmap
    regex: __meta_kubernetes_service_label_(.+)
  - action: replace
    source_labels:
    - __meta_kubernetes_namespace
    target_label: kubernetes_namespace
  - action: replace
    source_labels:
    - __meta_kubernetes_service_name
    target_label: kubernetes_name
  - action: replace
    source_labels:
    - __meta_kubernetes_pod_node_name
    target_label: kubernetes_node
  scrape_interval: 5m
  scrape_timeout: 30s
- honor_labels: true
  job_name: prometheus-pushgateway
  kubernetes_sd_configs:
  - role: service
  relabel_configs:
  - action: keep
    regex: pushgateway
    source_labels:
    - __meta_kubernetes_service_annotation_prometheus_io_probe
- job_name: kubernetes-services
  kubernetes_sd_configs:
  - role: service
  metrics_path: /probe
  params:
    module:
    - http_2xx
  relabel_configs:
  - action: keep
    regex: true
    source_labels:
    - __meta_kubernetes_service_annotation_prometheus_io_probe
  - source_labels:
    - __address__
    target_label: __param_target
  - replacement: blackbox
    target_label: __address__
  - source_labels:
    - __param_target
    target_label: instance
  - action: labelmap
    regex: __meta_kubernetes_service_label_(.+)
  - source_labels:
    - __meta_kubernetes_namespace
    target_label: kubernetes_namespace
  - source_labels:
    - __meta_kubernetes_service_name
    target_label: kubernetes_name
- job_name: kubernetes-pods
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - action: keep
    regex: true
    source_labels:
    - __meta_kubernetes_pod_annotation_prometheus_io_scrape
  - action: replace
    regex: (https?)
    source_labels:
    - __meta_kubernetes_pod_annotation_prometheus_io_scheme
    target_label: __scheme__
  - action: replace
    regex: (.+)
    source_labels:
    - __meta_kubernetes_pod_annotation_prometheus_io_path
    target_label: __metrics_path__
  - action: replace
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: $1:$2
    source_labels:
    - __address__
    - __meta_kubernetes_pod_annotation_prometheus_io_port
    target_label: __address__
  - action: labelmap
    regex: __meta_kubernetes_pod_label_(.+)
  - action: replace
    source_labels:
    - __meta_kubernetes_namespace
    target_label: kubernetes_namespace
  - action: replace
    source_labels:
    - __meta_kubernetes_pod_name
    target_label: kubernetes_pod_name
  - action: drop
    regex: Pending|Succeeded|Failed
    source_labels:
    - __meta_kubernetes_pod_phase
- job_name: kubernetes-pods-slow
  kubernetes_sd_configs:
  - role: pod
  relabel_configs:
  - action: keep
    regex: true
    source_labels:
    - __meta_kubernetes_pod_annotation_prometheus_io_scrape_slow
  - action: replace
    regex: (https?)
    source_labels:
    - __meta_kubernetes_pod_annotation_prometheus_io_scheme
    target_label: __scheme__
  - action: replace
    regex: (.+)
    source_labels:
    - __meta_kubernetes_pod_annotation_prometheus_io_path
    target_label: __metrics_path__
  - action: replace
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: $1:$2
    source_labels:
    - __address__
    - __meta_kubernetes_pod_annotation_prometheus_io_port
    target_label: __address__
  - action: labelmap
    regex: __meta_kubernetes_pod_label_(.+)
  - action: replace
    source_labels:
    - __meta_kubernetes_namespace
    target_label: kubernetes_namespace
  - action: replace
    source_labels:
    - __meta_kubernetes_pod_name
    target_label: kubernetes_pod_name
  - action: drop
    regex: Pending|Succeeded|Failed
    source_labels:
    - __meta_kubernetes_pod_phase
  scrape_interval: 5m
  scrape_timeout: 30s

{{- with .CustomScrapeConfigs }}
#######################################################################
# custom scraping configurations

{{ . }}
{{- end }}
`
)
