/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package util

import (
	"fmt"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/metrics"
)

func RetryN(maxAttempts int, f func(attempt int) error) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = f(attempt)
		if err != nil {
			continue
		}
		return nil
	}

	return fmt.Errorf("function did not succeed after %d attempts: %w", maxAttempts, err)
}

// measuredRetryN wraps retryNAttempts with code that counts
// the executed number of attempts and the runtimes for each
// attempt.
func MeasuredRetryN(
	runtimeMetric *prometheus.GaugeVec,
	//nolint:interfacer
	attemptsMetric prometheus.Gauge,
	log *zap.SugaredLogger,
	maxAttempts int,
	f func(attempt int) error,
) func() error {
	return func() error {
		attempts := 0

		err := RetryN(maxAttempts, func(attempt int) error {
			attempts++
			metric := runtimeMetric.With(prometheus.Labels{"attempt": strconv.Itoa(attempt)})

			return metrics.MeasureTime(metric, log, func() error {
				return f(attempt)
			})
		})

		attemptsMetric.Set(float64(attempts))
		metrics.UpdateMetrics(log)

		return err
	}
}
