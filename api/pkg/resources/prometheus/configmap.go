package prometheus

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMap returns a ConfigMap containing the prometheus config for the supplied data
func ConfigMap(data *resources.TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	var cm *corev1.ConfigMap
	if existing != nil {
		cm = existing
	} else {
		cm = &corev1.ConfigMap{}
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	configBuffer := bytes.Buffer{}
	configTpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(prometheusConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prometheus config template: %v", err)
	}
	if err := configTpl.Execute(&configBuffer, data); err != nil {
		return nil, fmt.Errorf("failed to render prometheus config template: %v", err)
	}

	cm.Name = resources.PrometheusConfigConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Labels = resources.BaseAppLabel(name, nil)
	cm.Data["prometheus.yaml"] = configBuffer.String()
	cm.Data["rules.yaml"] = prometheusRules

	return cm, nil
}

const prometheusConfig = `global:
  evaluation_interval: 1m
  scrape_interval: 15s
  external_labels:
    cluster: "{{ .Cluster.Name }}"
    region: "{{ .SeedDC }}"
rule_files:
- "/etc/prometheus/config/rules.yaml"
scrape_configs:
- job_name: etcd
  scheme: https
  metrics_path: '/metrics'
  static_configs:
  - targets: ['etcd-0.etcd.{{ .Cluster.Status.NamespaceName }}.svc.cluster.local:2379','etcd-1.etcd.{{ .Cluster.Status.NamespaceName }}.svc.cluster.local:2379','etcd-2.etcd.{{ .Cluster.Status.NamespaceName }}.svc.cluster.local:2379']
  tls_config:
    ca_file: /etc/etcd/pki/client/ca.crt
    cert_file: /etc/etcd/pki/client/apiserver-etcd-client.crt
    key_file: /etc/etcd/pki/client/apiserver-etcd-client.key

{{- range $i, $e := until 2 }}
- job_name: 'pods-{{ $i }}'

  kubernetes_sd_configs:
  - role: pod
    namespaces:
      names:
      - "{{ $.Cluster.Status.NamespaceName }}"

  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_{{ $i }}_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_{{ $i }}_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_{{ $i }}_port]
    action: replace
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: $1:$2
    target_label: __address__
  - source_labels: [__meta_kubernetes_pod_label_role, __meta_kubernetes_pod_label_app]
    action: replace
    target_label: job
    separator: ''
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: namespace
  - source_labels: [__meta_kubernetes_pod_name]
    action: replace
    target_label: pod

{{- end }}
alerting:
  alertmanagers:
  - dns_sd_configs:
    - names:
      - 'alertmanager-kubermatic.monitoring.svc.cluster.local'
      type: A
      port: 9093
`
