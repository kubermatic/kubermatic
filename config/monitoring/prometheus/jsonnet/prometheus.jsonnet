(import './alerts/ark.libsonnet') +
(import './alerts/kubermatic.libsonnet') +
(import './alerts/machine-controller.libsonnet') +
(import './alerts/blacklist.libsonnet') +
(import 'kubernetes-mixin/mixin.libsonnet') +
(import 'prometheus/mixin.libsonnet') +
(import 'node_exporter/mixin.libsonnet') +
{
  local arrayHas(haystack, needle) = std.length([1 for name in haystack if needle == name]) > 0,
  local goodRule(rule) = !std.objectHas(rule, 'alert') || !arrayHas($.prometheusAlertBlacklist, rule.alert),

  _config+:: {
    namespace: 'monitoring',

    // Selectors are inserted between {} in Prometheus queries.
    cadvisorSelector: 'job="cadvisor"',
    kubeletSelector: 'job="kubelet"',
    kubeStateMetricsSelector: 'job="kube-state-metrics"',
    nodeExporterSelector: 'app="node-exporter"',
    notKubeDnsSelector: 'job!="dns"',
    kubeSchedulerSelector: 'job="scheduler"',
    kubeControllerManagerSelector: 'job="controller-manager"',
    kubeApiserverSelector: 'job="apiserver"',
    machineControllerSelector: 'job="machine-controller"',

    // We build alerts for the presence of all these jobs. Those are global running applications
    jobs+:: {
      Cadvisor: $._config.cadvisorSelector,
      Kubelet: $._config.kubeletSelector,
      KubermaticAPI: 'job="pods",namespace="kubermatic",role="kubermatic-api"',
      KubermaticControllerManager: 'job="pods",namespace="kubermatic",role="controller-manager"',
      KubernetesApiserver: $._config.kubeApiserverSelector,
      KubeStateMetrics: $._config.kubeStateMetricsSelector,
      // KubernetesControllerManager: $._config.kubeControllerManagerSelector,
      // KubernetesScheduler: $._config.kubeSchedulerSelector,
    },

    runbookURLPattern: 'https://docs.kubermatic.io/monitoring/runbook/#alert-%s',

    prometheus+:: {
      name: 'kubermatic',
      rules: $.prometheusRules + $.prometheusAlerts,
    },
  },

  output: {
    groups: [
      {
        name: group.name,
        rules: std.filter(goodRule, group.rules),
      }
      for group in $._config.prometheus.rules.groups
    ]
  }
}
