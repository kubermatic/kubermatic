//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package prometheus

import (
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
)

// prometheusConfigMap creates the configmap for prometheus.
func prometheusConfigMap() reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return Name, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Labels == nil {
				cm.Labels = make(map[string]string)
			}
			cm.Labels[common.NameLabel] = Name

			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cm.Data["alerting_rules.yml"] = "{}"
			cm.Data["alerts"] = "{}"
			cm.Data["recording_rules.yml"] = "{}"
			cm.Data["rules"] = "{}"
			cm.Data["prometheus.yml"] = prometheusConfig

			return cm, nil
		}
	}
}

const prometheusConfig = `
global:
  evaluation_interval: 5m
  scrape_interval: 1m
  scrape_timeout: 10s
rule_files:
  - /etc/config/recording_rules.yml
  - /etc/config/alerting_rules.yml
  - /etc/config/rules
  - /etc/config/alerts
scrape_configs:
  - honor_labels: true
    job_name: cluster_kubelet_volume_stats
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__=~"kubelet_volume_stats_capacity_bytes|kubelet_volume_stats_used_bytes", job="kubelet"}'
    relabel_configs:
      - action: keep
        regex: user
        source_labels:
          - __meta_kubernetes_service_label_cluster

  - honor_labels: true
    job_name: seed-controller-manager
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app_kubernetes_io_name]
        action: keep
        regex: kubermatic-seed-controller-manager
      - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
        action: replace
        regex: ([^:]+)(?::\d+)?;(\d+)
        replacement: $1:$2
        target_label: __address__
  - honor_labels: true
    job_name: cluster_node_cpu_usage_seconds_total
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="node_cpu_usage_seconds_total"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
  - honor_labels: true
    job_name: cluster_machine_cpu_cores
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="machine_cpu_cores"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
  - honor_labels: true
    job_name: cluster_machine_memory_bytes
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="machine_memory_bytes"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
  - honor_labels: true
    job_name: cluster_node_memory_working_set_bytes
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="node_memory_working_set_bytes"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
  - honor_labels: true
    job_name: cluster_container_cpu_usage_seconds_total
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="container_cpu_usage_seconds_total"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
  - honor_labels: true
    job_name: cluster_container_memory_working_set_bytes
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="container_memory_working_set_bytes"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
  - honor_labels: true
    job_name: machine_controller_machines_total
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="machine_controller_machines_total"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
  - honor_labels: true
    job_name: kube_node_info
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="kube_node_info"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
  - honor_labels: true
    job_name: container_cpu_usage_seconds_total
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="container_cpu_usage_seconds_total"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
  - honor_labels: true
    job_name: container_memory_working_set_bytes
    kubernetes_sd_configs:
      - role: endpoints
    metrics_path: /federate
    params:
      match[]:
        - '{__name__="container_memory_working_set_bytes"}'
    relabel_configs:
      - action: keep
        regex: user
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_label_cluster
      - action: keep
        regex: web
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_endpoint_port_name
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_namespace
        target_label: Namespace
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_pod_name
        target_label: pod
      - action: replace
        regex: (.*)
        replacement: $1
        separator: ;
        source_labels:
          - __meta_kubernetes_service_name
        target_label: service
      - action: replace
        regex: (.*)
        replacement: web
        separator: ;
        target_label: endpoint
    scheme: http
`
