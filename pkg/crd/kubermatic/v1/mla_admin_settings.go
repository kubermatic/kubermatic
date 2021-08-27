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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MLAAdminSetting is the object representing cluster-specific administrator settings
// for KKP user cluster MLA (monitoring, logging & alerting) stack.
type MLAAdminSetting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MLAAdminSettingSpec `json:"spec"`
}

// MLAAdminSettingSpec specifies the cluster-specific administrator settings
// for KKP user cluster MLA (monitoring, logging & alerting) stack.
type MLAAdminSettingSpec struct {
	// Cluster is the reference to the user cluster whose MLA settings are defined in this object.
	Cluster corev1.ObjectReference `json:"cluster"`
	// MonitoringRateLimits contains rate-limiting configuration for monitoring in the user cluster.
	MonitoringRateLimits *MonitoringRateLimitSettings `json:"monitoringRateLimits,omitempty"`
	// LoggingRateLimits contains rate-limiting configuration logging in the user cluster.
	LoggingRateLimits *LoggingRateLimitSettings `json:"loggingRateLimits,omitempty"`
}

// MonitoringRateLimitSettings contains rate-limiting configuration for monitoring in the user cluster.
type MonitoringRateLimitSettings struct {
	// IngestionRate represents the ingestion rate limit in samples per second (Cortex `ingestion_rate`).
	IngestionRate int32 `json:"ingestionRate,omitempty"`
	// IngestionBurstSize represents ingestion burst size in samples per second (Cortex `ingestion_burst_size`).
	IngestionBurstSize int32 `json:"ingestionBurstSize,omitempty"`
	// MaxSeriesPerMetric represents maximum number of series per metric (Cortex `max_series_per_metric`).
	MaxSeriesPerMetric int32 `json:"maxSeriesPerMetric,omitempty"`
	// MaxSeriesTotal represents maximum number of series per this user cluster (Cortex `max_series_per_user`).
	MaxSeriesTotal int32 `json:"maxSeriesTotal,omitempty"`

	// QueryRate represents  query request rate limit per second (nginx `rate` in `r/s`).
	QueryRate int32 `json:"queryRate,omitempty"`
	// QueryBurstSize represents query burst size in number of requests (nginx `burst`).
	QueryBurstSize int32 `json:"queryBurstSize,omitempty"`
	// MaxSamplesPerQuery represents maximum number of samples during a query (Cortex `max_samples_per_query`).
	MaxSamplesPerQuery int32 `json:"maxSamplesPerQuery,omitempty"`
	// MaxSeriesPerQuery represents maximum number of timeseries during a query (Cortex `max_series_per_query`).
	MaxSeriesPerQuery int32 `json:"maxSeriesPerQuery,omitempty"`
}

// LoggingRateLimitSettings contains rate-limiting configuration for logging in the user cluster.
type LoggingRateLimitSettings struct {
	// IngestionRate represents ingestion rate limit in requests per second (nginx `rate` in `r/s`).
	IngestionRate int32 `json:"ingestionRate,omitempty"`
	// IngestionBurstSize represents ingestion burst size in number of requests (nginx `burst`).
	IngestionBurstSize int32 `json:"ingestionBurstSize,omitempty"` //

	// QueryRate represents query request rate limit per second (nginx `rate` in `r/s`).
	QueryRate int32 `json:"queryRate,omitempty"`
	// QueryBurstSize represents query burst size in number of requests (nginx `burst`).
	QueryBurstSize int32 `json:"queryBurstSize,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MLAAdminSettingList specifies a list of administrtor settigs for KKP
// user cluster MLA (monitoring, logging & alerting) stack
type MLAAdminSettingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MLAAdminSetting `json:"items"`
}
