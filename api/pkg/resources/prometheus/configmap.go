package prometheus

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"gopkg.in/yaml.v2"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

type tlsConfig struct {
	CAFile   string `yaml:"ca_file"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// customizationData is the data available to custom scraping configs and rules,
// containing everything required to scrape resources.
type customizationData struct {
	Cluster                  *kubermaticv1.Cluster
	APIServerHost            string
	EtcdTLS                  tlsConfig
	ApiserverTLS             tlsConfig
	ScrapingAnnotationPrefix string
}

type configTemplateData struct {
	TemplateData          interface{}
	APIServerHost         string
	EtcdTLSConfig         string
	ApiserverTLSConfig    string
	CustomScrapingConfigs string
}

// ConfigMapCreator returns a ConfigMapCreator containing the prometheus config for the supplied data
func ConfigMapCreator(data *resources.TemplateData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.PrometheusConfigConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			cluster := data.Cluster()

			// prepare TLS config
			etcdTLS := tlsConfig{
				CAFile:   "/etc/etcd/pki/client/ca.crt",
				CertFile: "/etc/etcd/pki/client/apiserver-etcd-client.crt",
				KeyFile:  "/etc/etcd/pki/client/apiserver-etcd-client.key",
			}

			apiserverTLS := tlsConfig{
				CAFile:   "/etc/kubernetes/ca.crt",
				CertFile: "/etc/kubernetes/prometheus-client.crt",
				KeyFile:  "/etc/kubernetes/prometheus-client.key",
			}

			// get custom scraping configs and rules
			customData := &customizationData{
				Cluster:                  cluster,
				APIServerHost:            fmt.Sprintf("%s:%d", cluster.Address.InternalName, cluster.Address.Port),
				EtcdTLS:                  etcdTLS,
				ApiserverTLS:             apiserverTLS,
				ScrapingAnnotationPrefix: data.MonitoringScrapeAnnotationPrefix(),
			}

			customScrapingFile := data.InClusterPrometheusScrapingConfigsFile()
			customScrapingConfigs, err := loadTemplatedFile(customScrapingFile, customData)
			if err != nil {
				return nil, fmt.Errorf("failed to load custom scraping configs file %s: %v", customScrapingFile, err)
			}

			customRulesFile := data.InClusterPrometheusRulesFile()
			customRules, err := loadTemplatedFile(customRulesFile, customData)
			if err != nil {
				return nil, fmt.Errorf("failed to load custom rules file %s: %v", customRulesFile, err)
			}

			// prepare tls_config stanza
			etcdTLSYaml, err := yaml.Marshal(etcdTLS)
			if err != nil {
				return nil, fmt.Errorf("failed to encode etcd TLS config as YAML: %v", err)
			}

			apiserverTLSYaml, err := yaml.Marshal(apiserverTLS)
			if err != nil {
				return nil, fmt.Errorf("failed to encode apiserver TLS config as YAML: %v", err)
			}

			// prepare config template
			configData := &configTemplateData{
				TemplateData:          data,
				APIServerHost:         customData.APIServerHost,
				CustomScrapingConfigs: customScrapingConfigs,
				EtcdTLSConfig:         strings.TrimSpace(string(etcdTLSYaml)),
				ApiserverTLSConfig:    strings.TrimSpace(string(apiserverTLSYaml)),
			}

			config, err := renderTemplate(prometheusConfig, configData)
			if err != nil {
				return nil, fmt.Errorf("failed to render Prometheus config: %v", err)
			}

			// update ConfigMap
			cm.Labels = resources.BaseAppLabel(name, nil)

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cm.Data["prometheus.yaml"] = config

			if data.InClusterPrometheusDisableDefaultRules() {
				delete(cm.Data, "rules.yaml")
			} else {
				cm.Data["rules.yaml"] = prometheusRules
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

func loadTemplatedFile(file string, data *customizationData) (string, error) {
	if file == "" {
		return "", nil
	}

	content, err := ioutil.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("couldn't read file: %v", err)
	}

	return renderTemplate(string(content), data)
}

func renderTemplate(tpl string, data interface{}) (string, error) {
	t, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse as Go template: %v", err)
	}

	output := bytes.Buffer{}
	if err := t.Execute(&output, data); err != nil {
		return "", fmt.Errorf("failed to render template: %v", err)
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
{{- if not .TemplateData.InClusterPrometheusDisableDefaultScrapingConfigs }}
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
  - source_labels: [__meta_kubernetes_pod_annotation_{{ .TemplateData.MonitoringScrapeAnnotationPrefix }}_port]
    action: keep
    regex: \d+
  - source_labels: [__meta_kubernetes_pod_annotation_{{ .TemplateData.MonitoringScrapeAnnotationPrefix }}_path]
    regex: (.+)
    action: replace
    target_label: __metrics_path__
  - source_labels: [__meta_kubernetes_namespace, __meta_kubernetes_pod_name, __meta_kubernetes_pod_annotation_{{ .TemplateData.MonitoringScrapeAnnotationPrefix }}_port, __metrics_path__]
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
{{- end }}

{{- with .CustomScrapingConfigs }}
#######################################################################
# custom scraping configurations

{{ . }}
{{- end }}
`
