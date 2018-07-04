{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'machine-controller',
        rules: [
          {
            alert: 'MachineControllerTooManyErrors',
            expr: |||
              sum(rate(machine_controller_errors_total[5m])) by (namespace) > 0.01
            ||| % $._config,
            'for': '10m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Machine Controller in {{ $labels.namespace }} has too many errors in its loop.',
            },
          },
        ],
      },
    ],
  },
}
