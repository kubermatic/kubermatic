package metrics

import (
	"k8s.io/client-go/util/workqueue"

	"github.com/prometheus/client_golang/prometheus"
)

// Copied from https://github.com/kubernetes/kubernetes/blob/master/pkg/util/workqueue/prometheus/prometheus.go
// Package prometheus sets the workqueue DefaultMetricsFactory to produce
// prometheus metrics. To use this package, you just have to import it.

func init() {
	workqueue.SetProvider(prometheusMetricsProvider{})
}

type prometheusMetricsProvider struct{}

func (prometheusMetricsProvider) NewDepthMetric(name string) workqueue.GaugeMetric {
	depth := prometheus.NewGauge(prometheus.GaugeOpts{
		Subsystem: name,
		Name:      "depth",
		Help:      "Current depth of workqueue: " + name,
	})
	// Upstream has prometheus.Register here
	prometheus.MustRegister(depth)
	return depth
}

func (prometheusMetricsProvider) NewAddsMetric(name string) workqueue.CounterMetric {
	adds := prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: name,
		Name:      "adds",
		Help:      "Total number of adds handled by workqueue: " + name,
	})
	// Upstream has prometheus.Register here
	prometheus.MustRegister(adds)
	return adds
}

func (prometheusMetricsProvider) NewLatencyMetric(name string) workqueue.SummaryMetric {
	latency := prometheus.NewSummary(prometheus.SummaryOpts{
		Subsystem: name,
		Name:      "queue_latency",
		Help:      "How long an item stays in workqueue" + name + " before being requested.",
	})
	// Upstream has prometheus.Register here
	prometheus.MustRegister(latency)
	return latency
}

func (prometheusMetricsProvider) NewWorkDurationMetric(name string) workqueue.SummaryMetric {
	workDuration := prometheus.NewSummary(prometheus.SummaryOpts{
		Subsystem: name,
		Name:      "work_duration",
		Help:      "How long processing an item from workqueue" + name + " takes.",
	})
	// Upstream has prometheus.Register here
	prometheus.MustRegister(workDuration)
	return workDuration
}

func (prometheusMetricsProvider) NewUnfinishedWorkSecondsMetric(name string) workqueue.SettableGaugeMetric {
	unfinished := prometheus.NewGauge(prometheus.GaugeOpts{
		Subsystem: name,
		Name:      "unfinished_work_seconds",
		Help: "How many seconds of work " + name + " has done that " +
			"is in progress and hasn't been observed by work_duration. Large " +
			"values indicate stuck threads. One can deduce the number of stuck " +
			"threads by observing the rate at which this increases.",
	})
	// Upstream has prometheus.Register here
	prometheus.MustRegister(unfinished)
	return unfinished
}

func (prometheusMetricsProvider) NewLongestRunningProcessorMicrosecondsMetric(name string) workqueue.SettableGaugeMetric {
	unfinished := prometheus.NewGauge(prometheus.GaugeOpts{
		Subsystem: name,
		Name:      "longest_running_processor_microseconds",
		Help: "How many microseconds has the longest running " +
			"processor for " + name + " been running.",
	})
	// Upstream has prometheus.Register here
	prometheus.MustRegister(unfinished)
	return unfinished
}

func (prometheusMetricsProvider) NewRetriesMetric(name string) workqueue.CounterMetric {
	retries := prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: name,
		Name:      "retries",
		Help:      "Total number of retries handled by workqueue: " + name,
	})
	// Upstream has prometheus.Register here
	prometheus.MustRegister(retries)
	return retries
}
