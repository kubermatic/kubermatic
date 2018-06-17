package main

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"

	backupcontroller "github.com/kubermatic/kubermatic/api/pkg/controller/backup"
)

const (
	metricNamespace = "kubermatic"
)

// UpdateControllerMetrics holds metrics used by Update controller
type UpdateControllerMetrics struct {
	Workers metrics.Gauge
}

// NewUpdateControllerMetrics creates UpdateControllerMetrics
// with default values initialized, so metrics always show up.
func NewUpdateControllerMetrics() *UpdateControllerMetrics {
	subsystem := "update_controller"
	cm := &UpdateControllerMetrics{
		Workers: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running Update controller workers",
		}, nil),
	}

	cm.Workers.Set(0)
	return cm
}

// AddonControllerMetrics holds metrics used by Addon controller
type AddonControllerMetrics struct {
	Workers metrics.Gauge
}

// NewAddonControllerMetrics creates AddonControllerMetrics
// with default values initialized, so metrics always show up.
func NewAddonControllerMetrics() *AddonControllerMetrics {
	subsystem := "addon_controller"
	cm := &AddonControllerMetrics{
		Workers: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running addon controller workers",
		}, nil),
	}

	cm.Workers.Set(0)
	return cm
}

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
