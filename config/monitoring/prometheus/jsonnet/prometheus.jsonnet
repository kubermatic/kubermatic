local k = import 'ksonnet/ksonnet.beta.3/k.libsonnet';

(import 'kubernetes-mixin/mixin.libsonnet') + {
  _config+:: {
    namespace: 'monitoring',

    // Selectors are inserted between {} in Prometheus queries.
    cadvisorSelector: 'job="cadvisor"',
    kubeletSelector: 'job="kubelet"',
    kubeStateMetricsSelector: 'job="kube-state-metrics"',
    nodeExporterSelector: 'app="node-exporter"',
    notKubeDnsSelector: 'job!="kube-dns"',
    kubeSchedulerSelector: 'job="kube-scheduler"',
    kubeControllerManagerSelector: 'job="kube-controller-manager"',
    kubeApiserverSelector: 'job="apiserver"',

    // We build alerts for the presence of all these jobs.
    jobs+:: {
      Cadvisor: $._config.cadvisorSelector,
      Kubelet: $._config.kubeletSelector,
      KubernetesApiserver: $._config.kubeApiserverSelector,
      KubeStateMetrics: $._config.kubeStateMetricsSelector,
      // KubernetesControllerManager: $._config.kubeControllerManagerSelector,
      // KubernetesScheduler: $._config.kubeSchedulerSelector,
    },

    prometheus+:: {
      name: 'kubermatic',
      rules: $.prometheusRules + $.prometheusAlerts,
    },
  },

  prometheus+:: {
    rules+:
      $._config.prometheus.rules,
  },
}
