(import './alerts/machine-controller.libsonnet') +
(import 'etcd-mixin/mixin.libsonnet') +
{
  _config+:: {
    runbookURLPattern: 'https://docs.kubermatic.io/monitoring/runbook/#alert-%s',

    prometheus+:: {
      name: 'kubermatic',
      rules: $.prometheusAlerts,
    },
  },

  prometheus+:: {
    rules+:
      $._config.prometheus.rules,
  },
}
