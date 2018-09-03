{
  prometheusAlerts+:: {
    groups+: [
      {
        name: 'ark',
        rules: [
          {
            alert: 'ArkBackupTakesTooLong',
            expr: '(ark_backup_attempt_total - ark_backup_success_total) > 0',
            'for': '60m',
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'Backup schedule {{ $labels.schedule }} has been taking more than 60min already.',
            },
          },
          {
            alert: 'ArkNoRecentBackup',
            expr: 'changes(ark_backup_success_total[25h]) < 1', // 25 hours to allow for one hour backup runtime
            labels: {
              severity: 'warning',
            },
            annotations: {
              message: 'There has not been a successful backup for schedule {{ $labels.schedule }} in the last 24 hours.',
            },
          },
        ],
      },
    ],
  },
}
