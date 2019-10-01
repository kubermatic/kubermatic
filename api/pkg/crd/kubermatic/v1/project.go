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

//+genclient
//+genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Project is the type describing a project.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec"`
	Status ProjectStatus `json:"status"`
}

// ProjectSpec is a specification of a project.
type ProjectSpec struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
}

// ProjectStatus represents the current status of a project.
type ProjectStatus struct {
	Phase string `json:"phase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ProjectList is a collection of projects.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Project `json:"items"`
}
