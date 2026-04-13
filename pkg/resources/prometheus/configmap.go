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
	// RemoteWriteConfig is the pre-rendered remote_write YAML block, generated
	// dynamically based on the number of seed Prometheus replicas at reconcile time.
	RemoteWriteConfig string
}

// writeRelabelRegex is the keep regex for write_relabel_configs. It covers exactly
// the metrics referenced by rules in usercluster-monitoring.yaml.
// etcd_.* covers all etcd_server/disk/network/mvcc/debugging metrics.
// machine_controller_.* and machine_deployment_.* cover machine-controller.
// konnectivity_.* and envoy_.* are no-ops for clusters not using those features.
const writeRelabelRegex = `up|process_open_fds|process_max_fds|process_resident_memory_bytes|process_cpu_seconds_total|etcd_.*|grpc_server_handling_seconds_bucket|apiserver_storage_size_bytes|apiserver_admission_webhook_admission_latencies_seconds_count|machine_controller_.*|machine_deployment_.*|kube_node_status_condition|kube_node_info|konnectivity_network_proxy_server_.*|envoy_cluster_upstream_.*`

// buildRemoteWriteConfig generates the remote_write YAML block targeting each seed
// Prometheus replica pod directly via StatefulSet headless DNS. Writing to every
// pod individually ensures each replica holds the full dataset — equivalent to the
// old /federate pull model where both replicas independently scraped /federate.
//
// Returns an empty string when replicas is 0 (seed monitoring not installed).
// In that case the agent still scrapes but discards immediately — no WAL buildup.
func buildRemoteWriteConfig(replicas int32) string {
	if replicas <= 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("remote_write:\n")
	for i := int32(0); i < replicas; i++ {
		fmt.Fprintf(&sb, "- url: http://prometheus-%d.prometheus.monitoring.svc.cluster.local:9090/api/v1/write\n", i)
		fmt.Fprintf(&sb, "  name: seed-prometheus-%d\n", i)
		sb.WriteString("  write_relabel_configs:\n")
		sb.WriteString("  - source_labels: [__name__]\n")
		sb.WriteString("    action: keep\n")
		fmt.Fprintf(&sb, "    regex: '%s'\n", writeRelabelRegex)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// ConfigMapReconciler returns a ConfigMapReconciler containing the prometheus config for the supplied data.
// seedPrometheusReplicas is the current replica count of the seed Prometheus StatefulSet; it is used
// to generate one remote_write target per replica pod so every replica receives the full dataset.
func ConfigMapReconciler(data *resources.TemplateData, seedPrometheusReplicas int32) reconciling.NamedConfigMapReconcilerFactory {
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
				RemoteWriteConfig:        buildRemoteWriteConfig(seedPrometheusReplicas),
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

			// Agent mode cannot evaluate rules; alerting rules are now managed at the
			// seed-level Prometheus. Remove any previously deployed rules files.
			delete(cm.Data, "rules.yaml")
			delete(cm.Data, "rules-custom.yaml")

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
  scrape_interval: 30s
  external_labels:
    cluster: "{{ .TemplateData.Cluster.Name }}"
    seed_cluster: "{{ .TemplateData.Seed.Name }}"

# Push scraped metrics to all seed-level Prometheus replicas directly.
# Writing to each pod via the StatefulSet headless DNS (not the ClusterIP Service)
# ensures every replica receives the full dataset — equivalent to the old pull
# federation model where both replicas independently scraped /federate.
# This gives Thanos deduplication identical timestamps across replicas (better than
# federation where each replica scraped at a slightly different moment).
#
# write_relabel_configs keeps only the metrics referenced by alerting and recording
# rules in charts/monitoring/prometheus/rules/src/kubermatic-seed/usercluster-monitoring.yaml,
# matching the cardinality of the old /federate filter ({kubermatic="federate"}).
# Replica count and targets are generated at reconcile time from the seed Prometheus StatefulSet.
{{ .RemoteWriteConfig }}

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
