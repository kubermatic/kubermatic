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
