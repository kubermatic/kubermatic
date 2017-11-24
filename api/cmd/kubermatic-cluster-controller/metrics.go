package main

import (
	"github.com/go-kit/kit/metrics"
)

type ControllerMetrics struct {
	cluster metrics.Gauge
}
