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

const (
	// UserProjectBindingResourceName represents "Resource" defined in Kubernetes.
	UserProjectBindingResourceName = "userprojectbindings"

	// UserProjectBindingKind represents "Kind" defined in Kubernetes.
	UserProjectBindingKind = "UserProjectBinding"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".spec.projectID",name="ProjectID",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.group",name="Group",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.userEmail",name="UserEmail",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// UserProjectBinding specifies a binding between a user and a project
// This resource is used by the user management to manipulate members of the given project.
type UserProjectBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes a KKP user and project binding.
	Spec UserProjectBindingSpec `json:"spec,omitempty"`
}

// UserProjectBindingSpec specifies a user and project binding.
type UserProjectBindingSpec struct {
	// UserEmail is the email of the user that is bound to the given project.
	UserEmail string `json:"userEmail"`
	// ProjectID is the name of the target project.
	ProjectID string `json:"projectID"`

	// TODO: add "Role" field and deprecate "Group" in favour of it to be in line with GroupProjectBinding resource.

	// Group is the user's group, determining their permissions within the project.
	// Must be one of `owners`, `editors`, `viewers` or `projectmanagers`.
	Group string `json:"group"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// UserProjectBindingList is a list of KKP user and project bindings.
type UserProjectBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is the list of KKP user and project bindings.
	Items []UserProjectBinding `json:"items"`
}
