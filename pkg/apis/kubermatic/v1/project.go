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

// These are the valid phases of a project.
const (
	// ProjectActive means the project is available for use in the system
	ProjectActive string = "Active"

	// ProjectInactive means the project is inactive and requires further initialization
	ProjectInactive string = "Inactive"

	// ProjectTerminating means the project is undergoing graceful termination
	ProjectTerminating string = "Terminating"
)

const (
	// ProjectResourceName represents "Resource" defined in Kubernetes
	ProjectResourceName = "projects"

	// ProjectKindName represents "Kind" defined in Kubernetes
	ProjectKindName = "Project"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Project is the type describing a project.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// ProjectSpec is a specification of a project.
type ProjectSpec struct {
	Name string `json:"name"`
}

// ProjectStatus represents the current status of a project.
type ProjectStatus struct {
	Phase string `json:"phase"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ProjectList is a collection of projects.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Project `json:"items"`
}
