package main

import (
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/metrics/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

const (
	metricNamespace = "kubermatic"
)

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
