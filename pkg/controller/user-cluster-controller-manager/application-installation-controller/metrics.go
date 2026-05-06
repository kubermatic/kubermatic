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

package applicationinstallationcontroller

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	metricsNamespace = "kkp"
	metricsSubsystem = "application_installation"
)

var (
	// applicationInstallationFailures tracks the number of failures for each ApplicationInstallation.
	// This metric can be used to alert when an application is approaching or has exceeded the retry limit.
	applicationInstallationFailures = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "failures",
			Help:      "Number of installation/upgrade failures for an ApplicationInstallation",
		},
		[]string{"namespace", "name", "application"},
	)

	// applicationInstallationStuck indicates whether an ApplicationInstallation is stuck (1) or not (0).
	// An application is considered stuck when it has exceeded the maximum retry limit.
	applicationInstallationStuck = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "stuck",
			Help:      "Whether an ApplicationInstallation is stuck (1=stuck, 0=ok). An application is stuck when it has exceeded the maximum retry limit.",
		},
		[]string{"namespace", "name", "application"},
	)

	// applicationInstallationReady indicates the ready status of an ApplicationInstallation.
	// Values: 1 = ready (successfully installed), 0 = not ready (failed or in progress), -1 = unknown.
	applicationInstallationReady = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "ready",
			Help:      "Ready status of an ApplicationInstallation (1=ready, 0=not ready, -1=unknown)",
		},
		[]string{"namespace", "name", "application"},
	)

	// applicationInstallationLastSuccessTimestamp records the timestamp of the last successful installation/upgrade.
	applicationInstallationLastSuccessTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: metricsSubsystem,
			Name:      "last_success_timestamp_seconds",
			Help:      "Unix timestamp of the last successful installation/upgrade",
		},
		[]string{"namespace", "name", "application"},
	)
)

func init() {
	// Register metrics with the controller-runtime metrics registry
	metrics.Registry.MustRegister(
		applicationInstallationFailures,
		applicationInstallationStuck,
		applicationInstallationReady,
		applicationInstallationLastSuccessTimestamp,
	)
}

// updateMetrics updates the Prometheus metrics for an ApplicationInstallation.
func updateMetrics(namespace, name, application string, failures int, isStuck bool, readyStatus int, lastSuccessTime float64) {
	applicationInstallationFailures.WithLabelValues(namespace, name, application).Set(float64(failures))

	stuckValue := 0.0
	if isStuck {
		stuckValue = 1.0
	}
	applicationInstallationStuck.WithLabelValues(namespace, name, application).Set(stuckValue)

	applicationInstallationReady.WithLabelValues(namespace, name, application).Set(float64(readyStatus))

	if lastSuccessTime > 0 {
		applicationInstallationLastSuccessTimestamp.WithLabelValues(namespace, name, application).Set(lastSuccessTime)
	}
}

// deleteMetrics removes the Prometheus metrics for an ApplicationInstallation when it's deleted.
func deleteMetrics(namespace, name, application string) {
	applicationInstallationFailures.DeleteLabelValues(namespace, name, application)
	applicationInstallationStuck.DeleteLabelValues(namespace, name, application)
	applicationInstallationReady.DeleteLabelValues(namespace, name, application)
	applicationInstallationLastSuccessTimestamp.DeleteLabelValues(namespace, name, application)
}
