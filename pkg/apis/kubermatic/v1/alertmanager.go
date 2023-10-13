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

const (
	// AlertmanagerResourceName represents "Resource" defined in Kubernetes.
	AlertmanagerResourceName = "alertmanagers"

	// AlertmanagerKindName represents "Kind" defined in Kubernetes.
	AlertmanagerKindName = "Alertmanager"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

type Alertmanager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the configuration of the Alertmanager.
	Spec AlertmanagerSpec `json:"spec,omitempty"`
	// Status stores status information about the Alertmanager.
	Status AlertmanagerStatus `json:"status,omitempty"`
}

// AlertmanagerSpec describes the configuration of the Alertmanager.
type AlertmanagerSpec struct {
	// ConfigSecret refers to the Secret in the same namespace as the Alertmanager object,
	// which contains configuration for this Alertmanager.
	ConfigSecret corev1.LocalObjectReference `json:"configSecret"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

type AlertmanagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items refers to the list of Alertmanager objects.
	Items []Alertmanager `json:"items"`
}

// AlertmanagerStatus stores status information about the AlertManager.
type AlertmanagerStatus struct {
	// ConfigStatus stores status information about the AlertManager configuration.
	ConfigStatus AlertmanagerConfigurationStatus `json:"configStatus,omitempty"`
}

// AlertmanagerConfigurationStatus stores status information about the AlertManager configuration.
type AlertmanagerConfigurationStatus struct {
	// LastUpdated stores the last successful time when the configuration was successfully applied
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
	// Status of whether the configuration was applied, one of True, False
	Status corev1.ConditionStatus `json:"status"`
	// ErrorMessage contains a default error message in case the configuration could not be applied.
	// Will be reset if the error was resolved and condition becomes True
	ErrorMessage string `json:"errorMessage,omitempty"`
}
