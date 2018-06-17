package main

import (
	"github.com/go-kit/kit/metrics/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"

	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/backup"
)

const (
	metricNamespace = "kubermatic"
)

// NewBackupControllerMetrics creates BackupControllerMetrics
// with default values initialized, so metrics always show up.
func NewBackupControllerMetrics() backupcontroller.Metrics {
	subsystem := "backup_controller"
	cm := backupcontroller.Metrics{
		Workers: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running backup controller workers",
		}, nil),
		CronJobCreationTimestamp: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "cronjob_creation_timestamp_seconds",
			Help:      "The timestamp at which a cronjob for a given cluster was created",
		}, []string{"cluster"}),
		CronJobUpdateTimestamp: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "cronjob_update_timestamp_seconds",
			Help:      "The timestamp at which a cronjob for a given cluster was last updated",
		}, []string{"cluster"}),
	}

	cm.Workers.Set(0)
	return cm
}
