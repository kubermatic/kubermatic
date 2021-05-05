/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"

	"k8s.io/client-go/util/workqueue"
)

// Copied from https://github.com/kubernetes/kubernetes/blob/master/pkg/util/workqueue/prometheus/prometheus.go
// and  https://github.com/kubernetes-sigs/controller-runtime/blob/v0.3.0/pkg/metrics/workqueue.go
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

func (prometheusMetricsProvider) NewLatencyMetric(queue string) workqueue.HistogramMetric {
	m := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        "workqueue_queue_duration_seconds",
		Help:        "How long in seconds an item stays in workqueue before being requested.",
		ConstLabels: prometheus.Labels{"name": queue},
		Buckets:     prometheus.ExponentialBuckets(10e-9, 10, 10),
	})
	prometheus.MustRegister(m)
	return m
}

func (prometheusMetricsProvider) NewWorkDurationMetric(queue string) workqueue.HistogramMetric {
	const name = "workqueue_work_duration_seconds"
	m := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:        name,
		Help:        "How long in seconds processing an item from workqueue takes.",
		ConstLabels: prometheus.Labels{"name": queue},
		Buckets:     prometheus.ExponentialBuckets(10e-9, 10, 10),
	})
	prometheus.MustRegister(m)
	return m
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

func (prometheusMetricsProvider) NewLongestRunningProcessorSecondsMetric(queue string) workqueue.SettableGaugeMetric {
	const name = "workqueue_longest_running_processor_seconds"
	m := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: name,
		Help: "How many seconds has the longest running " +
			"processor for workqueue been running.",
		ConstLabels: prometheus.Labels{"name": queue},
	})
	prometheus.MustRegister(m)
	return m
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

func (prometheusMetricsProvider) NewDeprecatedLongestRunningProcessorMicrosecondsMetric(queue string) workqueue.SettableGaugeMetric {
	m := prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "workqueue_longest_running_processor_microseconds",
		Help: "(Deprecated) How many microseconds has the longest running " +
			"processor for workqueue been running.",
		ConstLabels: prometheus.Labels{"name": queue},
	})
	prometheus.MustRegister(m)
	return m
}

func (prometheusMetricsProvider) NewDeprecatedDepthMetric(queue string) workqueue.GaugeMetric {
	return noopMetric{}
}

func (prometheusMetricsProvider) NewDeprecatedAddsMetric(queue string) workqueue.CounterMetric {
	return noopMetric{}
}

func (prometheusMetricsProvider) NewDeprecatedLatencyMetric(queue string) workqueue.SummaryMetric {
	return noopMetric{}
}

func (prometheusMetricsProvider) NewDeprecatedWorkDurationMetric(queue string) workqueue.SummaryMetric {
	return noopMetric{}
}

func (prometheusMetricsProvider) NewDeprecatedUnfinishedWorkSecondsMetric(queue string) workqueue.SettableGaugeMetric {
	return noopMetric{}
}

func (prometheusMetricsProvider) NewDeprecatedRetriesMetric(queue string) workqueue.CounterMetric {
	return noopMetric{}
}

type noopMetric struct{}

func (noopMetric) Inc() {}

func (noopMetric) Dec() {}

func (noopMetric) Set(float64) {}

func (noopMetric) Observe(float64) {}
