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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced

// ComponentSettings is the object representing component settings.
type ComponentSettings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ComponentSettingsSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// ComponentSettingsList contains a list of ComponentSettings
type ComponentSettingsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ComponentSettings `json:"items"`
}

type ComponentSettingsSpec struct {
	Apiserver         APIServerSettings       `json:"apiserver"`
	ControllerManager DeploymentSettings      `json:"controllerManager"`
	Scheduler         DeploymentSettings      `json:"scheduler"`
	Etcd              EtcdStatefulSetSettings `json:"etcd"`
	Prometheus        StatefulSetSettings     `json:"prometheus"`
}

type APIServerSettings struct {
	DeploymentSettings `json:",inline"`

	EndpointReconcilingDisabled *bool `json:"endpointReconcilingDisabled,omitempty"`
}

type DeploymentSettings struct {
	Replicas  *int32                       `json:"replicas,omitempty"`
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type StatefulSetSettings struct {
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type EtcdStatefulSetSettings struct {
	// +kubebuilder:validation:Minimum:=1
	// +kubebuilder:validation:Maximum:=9
	ClusterSize int                          `json:"clusterSize,omitempty"`
	Resources   *corev1.ResourceRequirements `json:"resources,omitempty"`
}
