package prometheus

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type promTplModel struct {
	TemplateData                       interface{}
	InClusterPrometheusScrapingConfigs string
}

// ConfigMap returns a ConfigMap containing the prometheus config for the supplied data
func ConfigMap(data resources.ConfigMapDataProvider, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	var cm *corev1.ConfigMap
	if existing != nil {
		cm = existing
	} else {
		cm = &corev1.ConfigMap{}
	}
	if cm.Data == nil {
		cm.Data = map[string]string{}
	}

	model := &promTplModel{TemplateData: data.TemplateData()}
	scrapingConfigsFile := data.InClusterPrometheusScrapingConfigsFile()
	if scrapingConfigsFile != "" {
		scrapingConfigs, err := ioutil.ReadFile(scrapingConfigsFile)
		if err != nil {
			return nil, fmt.Errorf("couldn't read custom scraping configs file, see: %v", err)
		}

		model.InClusterPrometheusScrapingConfigs = string(scrapingConfigs)
	}

	configBuffer := bytes.Buffer{}
	configTpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(prometheusConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prometheus config template: %v", err)
	}
	if err := configTpl.Execute(&configBuffer, model); err != nil {
		return nil, fmt.Errorf("failed to render prometheus config template: %v", err)
	}

	cm.Name = resources.PrometheusConfigConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Labels = resources.BaseAppLabel(name, nil)
	cm.Data["prometheus.yaml"] = configBuffer.String()

	if data.InClusterPrometheusDisableDefaultRules() {
		delete(cm.Data, "rules.yaml")
	} else {
		cm.Data["rules.yaml"] = prometheusRules
	}

	rulesFile := data.InClusterPrometheusRulesFile()
	if rulesFile == "" {
		delete(cm.Data, "rules-custom.yaml")
	} else {
		customRules, err := ioutil.ReadFile(rulesFile)
		if err != nil {
			return nil, fmt.Errorf("couldn't read custom rules file, see: %v", err)
		}

		cm.Data["rules-custom.yaml"] = string(customRules)
	}

	return cm, nil
}

const prometheusConfig = `global:
  evaluation_interval: 30s
  scrape_interval: 30s
  external_labels:
    cluster: "{{ .TemplateData.Cluster.Name }}"
    seed_cluster: "{{ .TemplateData.SeedDC }}"
rule_files:
- "/etc/prometheus/config/rules*.yaml"
scrape_configs:
{{- if .InClusterPrometheusScrapingConfigs }}
{{ .InClusterPrometheusScrapingConfigs }}
{{- end }}
{{- if not .TemplateData.InClusterPrometheusDisableDefaultScrapingConfigs }}
- job_name: etcd
  scheme: https
  metrics_path: '/metrics'
  static_configs:
  - targets:
    - 'etcd-0.etcd.{{ .TemplateData.Cluster.Status.NamespaceName }}.svc.cluster.local:2379'
    - 'etcd-1.etcd.{{ .TemplateData.Cluster.Status.NamespaceName }}.svc.cluster.local:2379'
    - 'etcd-2.etcd.{{ .TemplateData.Cluster.Status.NamespaceName }}.svc.cluster.local:2379'
  tls_config:
    ca_file: /etc/etcd/pki/client/ca.crt
    cert_file: /etc/etcd/pki/client/apiserver-etcd-client.crt
    key_file: /etc/etcd/pki/client/apiserver-etcd-client.key
  relabel_configs:
  - source_labels: [__address__]
    regex: (etcd-\d+).+
    action: replace
    replacement: $1
    target_label: instance

{{- range $i, $e := until 2 }}
- job_name: 'pods-{{ $i }}'

  kubernetes_sd_configs:
  - role: pod
    namespaces:
      names:
      - "{{ $.TemplateData.Cluster.Status.NamespaceName }}"

{{- if semverCompare ">=1.11.0, <= 1.11.3" $.TemplateData.Cluster.Spec.Version }}
  metric_relabel_configs:
  - source_labels: [job, __name__]
    regex: 'controller-manager;rest_.*'
    action: drop
{{- end }}

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
{{- end }}
alerting:
  alertmanagers:
  - dns_sd_configs:
    - names:
      - 'alertmanager-kubermatic.monitoring.svc.cluster.local'
      type: A
      port: 9093
`
