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

package prometheus

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

type TLSConfig struct {
	CAFile   string `yaml:"ca_file"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// CustomizationData is the data available to custom scraping configs and rules,
// containing everything required to scrape resources. This is a public interface
// and changes to this struct could break existing custom scrape/rule configs, so
// care must be taken when changing this.
type CustomizationData struct {
	Cluster                  *kubermaticv1.Cluster
	APIServerHost            string
	EtcdTLS                  TLSConfig
	ApiserverTLS             TLSConfig
	ScrapingAnnotationPrefix string
}

type configTemplateData struct {
	TemplateData          interface{}
	APIServerHost         string
	EtcdTLSConfig         string
	ApiserverTLSConfig    string
	CustomScrapingConfigs string
	// ScrapingAnnotationPrefix is normalized to fit into a Prometheus rewrite rule.
	ScrapingAnnotationPrefix string
}

// ConfigMapReconciler returns a ConfigMapReconciler containing the prometheus config for the supplied data.
func ConfigMapReconciler(data *resources.TemplateData) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return resources.PrometheusConfigConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			cluster := data.Cluster()
			kubermaticConfig := data.KubermaticConfiguration()

			// prepare TLS config
			etcdTLS := TLSConfig{
				CAFile:   "/etc/etcd/pki/client/ca.crt",
				CertFile: "/etc/etcd/pki/client/apiserver-etcd-client.crt",
				KeyFile:  "/etc/etcd/pki/client/apiserver-etcd-client.key",
			}

			apiserverTLS := TLSConfig{
				CAFile:   "/etc/kubernetes/ca.crt",
				CertFile: "/etc/kubernetes/prometheus-client.crt",
				KeyFile:  "/etc/kubernetes/prometheus-client.key",
			}

			// normalize the custom scraping prefix to be a valid YAML identifier
			scrapeAnnotationPrefix := strings.NewReplacer(".", "_", "/", "").Replace(kubermaticConfig.Spec.UserCluster.Monitoring.ScrapeAnnotationPrefix)

			// get custom scraping configs and rules
			customData := &CustomizationData{
				Cluster:                  cluster,
				APIServerHost:            cluster.Status.Address.InternalName,
				EtcdTLS:                  etcdTLS,
				ApiserverTLS:             apiserverTLS,
				ScrapingAnnotationPrefix: scrapeAnnotationPrefix,
			}

			customScrapingConfigs, err := renderTemplate(kubermaticConfig.Spec.UserCluster.Monitoring.CustomScrapingConfigs, customData)
			if err != nil {
				return nil, fmt.Errorf("custom scraping configuration could not be parsed as a Go template: %w", err)
			}

			customRules, err := renderTemplate(kubermaticConfig.Spec.UserCluster.Monitoring.CustomRules, customData)
			if err != nil {
				return nil, fmt.Errorf("custom scraping rules could not be parsed as a Go template: %w", err)
			}

			// prepare tls_config stanza
			etcdTLSYaml, err := yaml.Marshal(etcdTLS)
			if err != nil {
				return nil, fmt.Errorf("failed to encode etcd TLS config as YAML: %w", err)
			}

			apiserverTLSYaml, err := yaml.Marshal(apiserverTLS)
			if err != nil {
				return nil, fmt.Errorf("failed to encode apiserver TLS config as YAML: %w", err)
			}

			// prepare config template
			configData := &configTemplateData{
				TemplateData:             data,
				APIServerHost:            customData.APIServerHost,
				CustomScrapingConfigs:    customScrapingConfigs,
				EtcdTLSConfig:            strings.TrimSpace(string(etcdTLSYaml)),
				ApiserverTLSConfig:       strings.TrimSpace(string(apiserverTLSYaml)),
				ScrapingAnnotationPrefix: scrapeAnnotationPrefix,
			}

			config, err := renderTemplate(prometheusConfig, configData)
			if err != nil {
				return nil, fmt.Errorf("failed to render Prometheus config: %w", err)
			}

			// update ConfigMap
			cm.Labels = resources.BaseAppLabels(name, nil)

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cm.Data["prometheus.yaml"] = config

			if kubermaticConfig.Spec.UserCluster.Monitoring.DisableDefaultRules {
				delete(cm.Data, "rules.yaml")
			} else {
				cm.Data["rules.yaml"] = prometheusRules

				// deploy DNSResolverDownAlert rule only if Konnectivity is disabled
				// (custom DNS resolver in not deployed in Konnectivity setup)
				if !data.IsKonnectivityEnabled() {
					cm.Data["rules.yaml"] += prometheusRuleDNSResolverDownAlert
				} else {
					cm.Data["rules.yaml"] += prometheusRuleKonnectivity
				}

				if cluster.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
					cm.Data["rules.yaml"] += prometheusRuleEnvoyAgentFederation
				}
			}

			if customRules == "" {
				delete(cm.Data, "rules-custom.yaml")
			} else {
				cm.Data["rules-custom.yaml"] = customRules
			}

			// make sure all files end with exactly one empty line to prevent needless pod restarts
			for k, v := range cm.Data {
				cm.Data[k] = strings.TrimSpace(v) + "\n"
			}

			return cm, nil
		}
	}
}

func renderTemplate(tpl string, data interface{}) (string, error) {
	t, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse as Go template: %w", err)
	}

	output := bytes.Buffer{}
	if err := t.Execute(&output, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return strings.TrimSpace(output.String()), nil
}

const prometheusConfig = `
global:
  evaluation_interval: 30s
  scrape_interval: 30s
  external_labels:
    cluster: "{{ .TemplateData.Cluster.Name }}"
    seed_cluster: "{{ .TemplateData.Seed.Name }}"

rule_files:
- "/etc/prometheus/config/rules*.yaml"

alerting:
  alertmanagers:
  - dns_sd_configs:
    # configure the Seed's alertmanager for the user cluster
    - names:
      - 'alertmanager.monitoring.svc.cluster.local'
      type: A
      port: 9093

scrape_configs:
{{- if not .TemplateData.KubermaticConfiguration.Spec.UserCluster.Monitoring.DisableDefaultScrapingConfigs }}
#######################################################################
# These rules will scrape pods running inside the seed cluster.

# scrape the etcd pods
- job_name: etcd
  scheme: https
  tls_config:
{{ .EtcdTLSConfig | indent 4 }}

  static_configs:
  - targets:
    - 'etcd-0.etcd.{{ .TemplateData.Cluster.Status.NamespaceName }}.svc.cluster.local:2379'
    - 'etcd-1.etcd.{{ .TemplateData.Cluster.Status.NamespaceName }}.svc.cluster.local:2379'
    - 'etcd-2.etcd.{{ .TemplateData.Cluster.Status.NamespaceName }}.svc.cluster.local:2379'

  relabel_configs:
  - source_labels: [__address__]
    regex: (etcd-\d+).+
    action: replace
    replacement: $1
    target_label: instance

# scrape the cluster's control plane (apiserver, controller-manager, scheduler)
- job_name: kubernetes-control-plane
  scheme: https
  tls_config:
{{ .ApiserverTLSConfig | indent 4 }}
    # insecure_skip_verify is needed because the apiservers certificate
    # does not contain a common name for the pod's ip address
    insecure_skip_verify: true

  kubernetes_sd_configs:
  - role: pod
    namespaces:
      names:
      - "{{ .TemplateData.Cluster.Status.NamespaceName }}"

  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape_with_kube_cert]
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
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: namespace
  - source_labels: [__meta_kubernetes_pod_name]
    action: replace
    target_label: pod
  - source_labels: [__meta_kubernetes_pod_label_app]
    action: replace
    target_label: job

  # drop very expensive apiserver metrics
  metric_relabel_configs:
  - source_labels: [__name__]
    regex: 'apiserver_request_(duration|latencies)_.*'
    action: drop
  - source_labels: [__name__]
    regex: 'apiserver_response_sizes_.*'
    action: drop

# scrape other cluster control plane components, like kube-state-metrics, DNS resolver,
# machine-controller etcd.
- job_name: control-plane-pods
  kubernetes_sd_configs:
  - role: pod
    namespaces:
      names:
      - "{{ $.TemplateData.Cluster.Status.NamespaceName }}"

  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app, __meta_kubernetes_pod_container_init]
    regex: "kube-state-metrics;true"
    action: drop
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
    target_label: job
    separator: ''
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: namespace
  - source_labels: [__meta_kubernetes_pod_name]
    action: replace
    target_label: pod

{{- if .TemplateData.IsKonnectivityEnabled }}
# scrape Konnectivity server metrics from apiserver pods
- job_name: konnectivity-server
  scheme: http

  kubernetes_sd_configs:
  - role: pod
    namespaces:
      names:
      - "{{ $.TemplateData.Cluster.Status.NamespaceName }}"

  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app]
    action: keep
    regex: apiserver
  - source_labels: [__meta_kubernetes_pod_container_name]
    action: keep
    regex: konnectivity-server
  - source_labels: [__address__]
    action: replace
    regex: ([^:]+)(?::\d+)?
    replacement: $1:8133
    target_label: __address__
  - target_label: __metrics_path__
    replacement: /metrics
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: namespace
  - source_labels: [__meta_kubernetes_pod_name]
    action: replace
    target_label: pod
  - target_label: job
    replacement: konnectivity-server
{{- end }}

#######################################################################
# These rules will scrape pods running inside the user cluster itself.

# scrape node metrics
- job_name: nodes
  scheme: https
  tls_config:
{{ .ApiserverTLSConfig | indent 4 }}

  kubernetes_sd_configs:
  - role: node
    api_server: 'https://{{ .APIServerHost }}'
    tls_config:
{{ .ApiserverTLSConfig | indent 6 }}

  relabel_configs:
  - action: labelmap
    regex: __meta_kubernetes_node_label_(.+)
  - target_label: __address__
    replacement: '{{ .APIServerHost }}'
  - source_labels: [__meta_kubernetes_node_name]
    regex: (.+)
    target_label: __metrics_path__
    replacement: /api/v1/nodes/${1}/proxy/metrics

# scrape node cadvisor
- job_name: cadvisor
  scheme: https
  tls_config:
{{ .ApiserverTLSConfig | indent 4 }}

  kubernetes_sd_configs:
  - role: node
    api_server: 'https://{{ .APIServerHost }}'
    tls_config:
{{ .ApiserverTLSConfig | indent 6 }}

  relabel_configs:
  - action: labelmap
    regex: __meta_kubernetes_node_label_(.+)
  - target_label: __address__
    replacement: '{{ .APIServerHost }}'
  - source_labels: [__meta_kubernetes_node_name]
    regex: (.+)
    target_label: __metrics_path__
    replacement: /api/v1/nodes/${1}/proxy/metrics/cadvisor

# scrape pods inside the user cluster with a special annotation
- job_name: 'user-cluster-pods'
  scheme: https
  tls_config:
{{ .ApiserverTLSConfig | indent 4 }}

  kubernetes_sd_configs:
  - role: pod
    api_server: 'https://{{ .APIServerHost }}'
    tls_config:
{{ .ApiserverTLSConfig | indent 6 }}

  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_annotation_{{ .ScrapingAnnotationPrefix }}_port]
    action: keep
    regex: \d+
  - source_labels: [__meta_kubernetes_pod_annotation_{{ .ScrapingAnnotationPrefix }}_path]
    regex: (.+)
    action: replace
    target_label: __metrics_path__
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_name, __meta_kubernetes_pod_annotation_{{ .ScrapingAnnotationPrefix }}_port, __metrics_path__]
    action: replace
    regex: (.*);(.*);(.*);(.*)
    target_label: __metrics_path__
    replacement: /api/v1/namespaces/${1}/pods/${2}:${3}/proxy${4}
  - target_label: __address__
    replacement: '{{ .APIServerHost }}'
  - source_labels: [__meta_kubernetes_namespace]
    action: replace
    target_label: namespace
  - source_labels: [__meta_kubernetes_pod_name]
    action: replace
    target_label: pod

# scrape kubelet resources
- job_name: kubelet
  scheme: https
  tls_config:
{{ .ApiserverTLSConfig | indent 4 }}

  kubernetes_sd_configs:
  - role: node
    api_server: 'https://{{ .APIServerHost }}'
    tls_config:
{{ .ApiserverTLSConfig | indent 6 }}

  relabel_configs:
  - action: labelmap
    regex: __meta_kubernetes_node_label_(.+)
  - target_label: __address__
    replacement: '{{ .APIServerHost }}'
  - source_labels: [__meta_kubernetes_node_name]
    regex: (.+)
    target_label: __metrics_path__
    replacement: /api/v1/nodes/${1}/proxy/metrics

- job_name: resources
  scheme: https
  tls_config:
{{ .ApiserverTLSConfig | indent 4 }}

  kubernetes_sd_configs:
  - role: node
    api_server: 'https://{{ .APIServerHost }}'
    tls_config:
{{ .ApiserverTLSConfig | indent 6 }}

  relabel_configs:
  - action: labelmap
    regex: __meta_kubernetes_node_label_(.+)
  - target_label: __address__
    replacement: '{{ .APIServerHost }}'
  - source_labels: [__meta_kubernetes_node_name]
    regex: (.+)
    target_label: __metrics_path__
    replacement: /api/v1/nodes/${1}/proxy/metrics/resource

{{ if eq .TemplateData.Cluster.Spec.ExposeStrategy "Tunneling" -}}
# scrape envoy-agent
- job_name: 'envoy-agent'
  scheme: http
  tls_config:
{{ .ApiserverTLSConfig | indent 4 }}

  kubernetes_sd_configs:
  - role: pod
    api_server: 'https://{{ .APIServerHost }}'
    tls_config:
{{ .ApiserverTLSConfig | indent 6 }}

  relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app]
    regex: "envoy-agent"
    action: keep
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
{{- with .CustomScrapingConfigs -}}
#######################################################################
# custom scraping configurations

{{ . }}
{{- end }}
`
