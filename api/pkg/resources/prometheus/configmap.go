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
	cm.Labels = resources.GetLabels(name)
	cm.Data = map[string]string{
		"prometheus.yaml": configBuffer.String(),
		"rules.yaml":      prometheusRules,
	}

	return cm, nil
}

const (
	prometheusConfig = `
global:
  evaluation_interval: 30s
  scrape_interval: 30s
  external_labels:
    cluster: "{{ .Cluster.Name }}"
rule_files:
- "/etc/prometheus/config/rules.yaml"
scrape_configs:
- job_name: 'pods-0'
  kubernetes_sd_configs:
  - role: pod
    namespaces:
      names:
      - "{{ .Cluster.Status.NamespaceName }}"

  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
    action: keep
    regex: true
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
    action: replace
    target_label: __metrics_path__
    regex: (.+)
  - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
    action: replace
    regex: ([^:]+)(?::\d+)?;(\d+)
    replacement: $1:$2
    target_label: __address__
  - source_labels: [__meta_kubernetes_pod_label_role, __meta_kubernetes_pod_label_app]
    action: replace
    target_label: role
  - source_labels: [__meta_kubernetes_pod_label_release]
    action: replace
    target_label: release
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: kubernetes_namespace
  - source_labels: [__meta_kubernetes_pod_name]
    action: replace
    target_label: kubernetes_pod_name
alerting:
  alertmanagers: []
`

	prometheusRules = ``
)
