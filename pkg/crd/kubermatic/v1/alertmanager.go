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
	// AlertmanagerResourceName represents "Resource" defined in Kubernetes
	AlertmanagerResourceName = "alertmanagers"

	// AlertmanagerKindName represents "Kind" defined in Kubernetes
	AlertmanagerKindName = "Alertmanager"
)

//+genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Alertmanager struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AlertmanagerSpec   `json:"spec,omitempty"`
	Status AlertManagerStatus `json:"status,omitempty"`
}

type AlertmanagerSpec struct {
	// ConfigSecret refers to the Secret in the same namespace as the Alertmanager object,
	// which contains configuration for this Alertmanager.
	ConfigSecret corev1.LocalObjectReference `json:"configSecret"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AlertmanagerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Alertmanager `json:"items"`
}

// AlertManagerStatus stores status information about the AlertManager
type AlertManagerStatus struct {
	ConfigStatus AlertManagerConfigurationStatus `json:"configStatus,omitempty"`
}

// AlertManagerConfigurationStatus stores status information about the AlertManager configuration
type AlertManagerConfigurationStatus struct {
	// LastUpdated stores the last successful time when the configuration was successfully applied
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
	// Status of whether the configuration was applied, one of True, False
	Status corev1.ConditionStatus `json:"status"`
	// ErrorMessage contains a default error message in case the configuration could not be applied.
	// Will be reset if the error was resolved and condition becomes True
	ErrorMessage string `json:"errorMessage,omitempty"`
}
