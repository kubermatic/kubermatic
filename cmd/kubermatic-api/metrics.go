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

package main

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
)

var metrics = common.ServerMetrics{
	HTTPRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Count of all HTTP requests",
	}, []string{"code", "method"}),
	HTTPRequestsDuration: prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "A histogram of latencies for requests.",
			Buckets: []float64{.005, .01, .025, .05, 0.1, 0.25, 0.5, 1, 1.25, 1.85, 2, 5},
		},
		[]string{"method", "route"},
	),
	InitNodeDeploymentFailures: prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubermatic_api_init_node_deployment_failures",
			Help: "The number of times initial node deployment couldn't be created within the timeout",
		},
		[]string{"cluster", "datacenter"},
	),
}

// registerMetrics registers metrics for the API.
func registerMetrics() {
	prometheus.MustRegister(metrics.HTTPRequestsTotal)
	prometheus.MustRegister(metrics.HTTPRequestsDuration)
	prometheus.MustRegister(metrics.InitNodeDeploymentFailures)
}

// RouteLookupFunc is a delegate for getting a unique identifier for the route which matches the passed request.
type RouteLookupFunc func(*http.Request) string

// instrumentHandler wraps the passed handler with prometheus duration and counter tracking.
func instrumentHandler(next http.Handler, lookupRoute RouteLookupFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		promhttp.InstrumentHandlerCounter(metrics.HTTPRequestsTotal, next).ServeHTTP(w, r)
		metrics.HTTPRequestsDuration.With(prometheus.Labels{"route": lookupRoute(r), "method": r.Method}).Observe(time.Since(start).Seconds())
	}
}
