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

// ClusterControllerMetrics is a struct of all metrics used in
// the cluster controller.
type ClusterControllerMetrics struct {
	Clusters        metrics.Gauge
	ClusterPhases   metrics.Gauge
	Workers         metrics.Gauge
	UnhandledErrors metrics.Counter
}

// NewClusterControllerMetrics creates new ClusterControllerMetrics
// with default values initialized, so metrics always show up.
func NewClusterControllerMetrics() *ClusterControllerMetrics {
	subsystem := "cluster_controller"

	cm := &ClusterControllerMetrics{
		Clusters: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "clusters",
			Help:      "The number of currently managed clusters",
		}, nil),
		ClusterPhases: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "cluster_status_phase",
			Help:      "All phases a cluster can be in. 1 if the cluster is in that phase",
		}, []string{"cluster", "phase"}),
		Workers: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running cluster controller workers",
		}, nil),
		UnhandledErrors: prometheus.NewCounterFrom(prom.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "unhandled_errors_total",
			Help:      "The number of unhandled errors that occurred in the controller's reconciliation loop",
		}, nil),
	}

	// Set default values, so that these metrics always show up
	cm.Clusters.Set(0)
	cm.Workers.Set(0)

	return cm
}

// RBACGeneratorControllerMetrics holds metrics used by
// RBACGenerator controller
type RBACGeneratorControllerMetrics struct {
	Workers metrics.Gauge
}

// NewRBACGeneratorControllerMetrics creates RBACGeneratorControllerMetrics
// with default values initialized, so metrics always show up.
func NewRBACGeneratorControllerMetrics() *RBACGeneratorControllerMetrics {
	subsystem := "rbac_generator_controller"
	cm := &RBACGeneratorControllerMetrics{
		Workers: prometheus.NewGaugeFrom(prom.GaugeOpts{
			Namespace: metricNamespace,
			Subsystem: subsystem,
			Name:      "workers",
			Help:      "The number of running RBACGenerator controller workers",
		}, nil),
	}

	cm.Workers.Set(0)
	return cm
}

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
