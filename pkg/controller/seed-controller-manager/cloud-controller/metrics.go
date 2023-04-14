/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package cloud

import "github.com/prometheus/client_golang/prometheus"

var (
	totalProviderReconciliations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "kubermatic",
		Subsystem: "cloud_controller",
		Name:      "provider_reconciliations_total",
		Help:      "The total number of provider reconciliations for a usercluster",
	}, []string{"cluster", "provider"})

	successfulProviderReconciliations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "kubermatic",
		Subsystem: "cloud_controller",
		Name:      "provider_successful_reconciliations_total",
		Help:      "The number of successful provider reconciliations for a usercluster",
	}, []string{"cluster", "provider"})
)

func MustRegisterMetrics(c prometheus.Registerer) {
	c.MustRegister(totalProviderReconciliations)
	c.MustRegister(successfulProviderReconciliations)
}
