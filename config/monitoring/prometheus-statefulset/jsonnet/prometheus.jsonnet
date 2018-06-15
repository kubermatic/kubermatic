local k = import 'ksonnet/ksonnet.beta.3/k.libsonnet';

(import 'kubernetes-mixin/mixin.libsonnet') + {
  _config+:: {
    namespace: 'monitoring',

    // kubeStateMetricsSelector: 'job="kube-state-metrics"',
    // cadvisorSelector: 'job="kubernetes-cadvisor"',
    // nodeExporterSelector: 'job="kubernetes-node-exporter"',
    // kubeletSelector: 'job="kubernetes-kubelet"',

    jobs+:: {
        // Alertmanager: 'job="alertmanager-metalmatze"',
        KubeAPI: 'job="apiserver"',
        // Kubelet: $._config.kubeletSelector,
        // KubeStateMetrics: $._config.kubeStateMetricsSelector,
        // NodeExporter: $._config.nodeExporterSelector,
        // PrometheusOperator: 'job="prometheus-operator"',
        // PrometehusMetalMatze: 'job="prometheus-metalmatze"',
    },

    prometheus+:: {
      name: 'testing',
      rules: $.prometheusRules + $.prometheusAlerts,
    },
  },

  prometheus+:: {
    rules+:
      $._config.prometheus.rules,
  },
}
