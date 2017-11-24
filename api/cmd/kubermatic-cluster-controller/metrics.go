package main

import (
	"github.com/go-kit/kit/metrics"
)

// ControllerMetrics has all controller metrics
type ControllerMetrics struct {
	cluster metrics.Gauge
}
