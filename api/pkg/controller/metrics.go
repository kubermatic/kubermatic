package controller

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/apimachinery/pkg/util/runtime"
)

const (
	clusterControllerSubsystem = "kubermatic_controller_manager"

	errorsStatisticKey = "unhandled_errors_total"
)

var (
	errors = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Subsystem: clusterControllerSubsystem,
			Name:      errorsStatisticKey,
			Help:      "The number of unhandled errors that occurred in all controllers",
		},
	)
)

var registerMetrics sync.Once

// Register the metrics that are to be monitored.
func Register() {
	registerMetrics.Do(func() {
		prometheus.MustRegister(errors)
		errors.Set(0)

		// register error handler that will increment a counter that will be scraped by prometheus,
		// that accounts for all errors reported via a call to runtime.HandleError
		runtime.ErrorHandlers = append(runtime.ErrorHandlers, func(err error) {
			errors.Inc()
		})
	})
}
