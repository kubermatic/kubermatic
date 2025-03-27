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
	"k8c.io/kubermatic/sdk/v2/semver"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// AdmissionPluginList is the type representing a AdmissionPluginList.
type AdmissionPluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items refers to the list of Admission Plugins
	Items []AdmissionPlugin `json:"items"`
}

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// AdmissionPlugin is the type representing a AdmissionPlugin.
type AdmissionPlugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes an admission plugin name and in which k8s version it is supported.
	Spec AdmissionPluginSpec `json:"spec,omitempty"`
}

// AdmissionPluginSpec specifies admission plugin name and from which k8s version is supported.
type AdmissionPluginSpec struct {
	PluginName string `json:"pluginName"`

	// FromVersion flag can be empty. It means the plugin fit to all k8s versions
	FromVersion *semver.Semver `json:"fromVersion,omitempty"`
}
