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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=Active;Inactive;Terminating

type ProjectPhase string

// These are the valid phases of a project.
const (
	// ProjectActive means the project is available for use in the system.
	ProjectActive ProjectPhase = "Active"

	// ProjectInactive means the project is inactive and requires further initialization.
	ProjectInactive ProjectPhase = "Inactive"

	// ProjectTerminating means the project is undergoing graceful termination.
	ProjectTerminating ProjectPhase = "Terminating"
)

const (
	// ProjectResourceName represents "Resource" defined in Kubernetes.
	ProjectResourceName = "projects"

	// ProjectKindName represents "Kind" defined in Kubernetes.
	ProjectKindName = "Project"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.name",name="HumanReadableName",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Status",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// Project is the type describing a project. A project is a collection of
// SSH keys, clusters and members. Members are assigned by creating UserProjectBinding
// objects.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the configuration of the project.
	Spec ProjectSpec `json:"spec,omitempty"`
	// Status holds the current status of the project.
	Status ProjectStatus `json:"status,omitempty"`
}

// ProjectSpec is a specification of a project.
type ProjectSpec struct {
	// Name is the human-readable name given to the project.
	Name string `json:"name"`
	// AllowedOperatingSystems defines a map of operating systems that can be used for the machines inside this project.
	AllowedOperatingSystems allowedOperatingSystems `json:"allowedOperatingSystems,omitempty"`
}

// ProjectStatus represents the current status of a project.
type ProjectStatus struct {
	// Phase describes the project phase. New projects are in the `Inactive`
	// phase; after being reconciled they move to `Active` and during deletion
	// they are `Terminating`.
	Phase ProjectPhase `json:"phase"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ProjectList is a collection of projects.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of the projects.
	Items []Project `json:"items"`
}
