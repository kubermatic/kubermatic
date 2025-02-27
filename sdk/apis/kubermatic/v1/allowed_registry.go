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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	// AllowedRegistryResourceName represents "Resource" defined in Kubernetes.
	AllowedRegistryResourceName = "allowedregistry"

	// AllowedRegistryKindName represents "Kind" defined in Kubernetes.
	AllowedRegistryKindName = "AllowedRegistry"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".spec.registryPrefix",name="RegistryPrefix",type="string",description="RegistryPrefix contains the prefix of the registry which will be allowed. User clusters will be able to deploy only images which are prefixed with one of the allowed image registry prefixes."
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// AllowedRegistry is the object representing an allowed registry.
type AllowedRegistry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the desired state for an allowed registry.
	Spec AllowedRegistrySpec `json:"spec,omitempty"`
}

// AllowedRegistrySpec specifies the data for allowed registry spec.
type AllowedRegistrySpec struct {
	// RegistryPrefix contains the prefix of the registry which will be allowed. User clusters will be able to deploy
	// only images which are prefixed with one of the allowed image registry prefixes.
	RegistryPrefix string `json:"registryPrefix"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// AllowedRegistryList specifies a list of allowed registries.
type AllowedRegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items refers to the list of the allowed registries.
	Items []AllowedRegistry `json:"items"`
}
