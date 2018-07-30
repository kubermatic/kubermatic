package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/apimachinery/pkg/util/runtime"
)

const (
	errorsStatisticKey = "unhandled_errors_total"
)

// RegisterRuntimErrorMetricCounter the metrics that are to be monitored.
func RegisterRuntimErrorMetricCounter(subsystem string, registry prometheus.Registerer) {
	errors := prometheus.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      errorsStatisticKey,
			Help:      "The number of unhandled errors",
		},
	)

	registry.MustRegister(errors)

	// register error handler that will increment a counter that will be scraped by prometheus,
	// that accounts for all errors reported via a call to runtime.HandleError
	runtime.ErrorHandlers = append(runtime.ErrorHandlers, func(err error) {
		errors.Inc()
	})
}
