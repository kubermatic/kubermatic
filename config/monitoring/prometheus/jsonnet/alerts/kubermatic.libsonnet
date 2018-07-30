{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'kubermatic',
        rules: [
          {
            alert: 'KubermaticTooManyUnhandledErrors',
            expr: |||
              sum(rate(kubermatic_controller_manager_unhandled_errors_total[5m])) > 0.01
            ||| % $._config,
            'for': '10m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Kubermatic controller manager in {{ $labels.namespace }} is experiencing too many errors.',
            },
          },
          {
            alert: 'KubermaticClusterDeletionTakesTooLong',
            expr: '(time() - kubermatic_cluster_deleted) > 30*60',
            'for': '0m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Cluster {{ $labels.cluster }} is stuck in deletion for more than 30min.',
            },
          },
        ],
      },
    ],
  },
}
