{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'kubermatic',
        rules: [
          {
            alert: 'KubermaticTooManyUnhandledErrors',
            expr: |||
              sum(rate(kubermatic_cluster_controller_unhandled_errors_total[5m])) > 0.01
            ||| % $._config,
            'for': '10m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Kubermatic Controller Manager in {{ $labels.namespace }} has too many errors in its loop.',
            },
          },
          {
            alert: 'KubermaticStuckClusterPhase',
            expr: |||
              kubermatic_cluster_controller_cluster_status_phase{phase="running"} == 0 and
              kubermatic_cluster_controller_cluster_status_phase{phase="deleting"} == 0
            ||| % $._config,
            'for': '10m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Kubermatic cluster {{ $labels.cluster }} is stuck in unexpected phase {{ $labels.phase }}.',
            },
          },
        ],
      },
    ],
  },
}
