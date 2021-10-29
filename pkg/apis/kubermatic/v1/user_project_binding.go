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

	// UserProjectBindingResourceName represents "Resource" defined in Kubernetes
	UserProjectBindingResourceName = "userprojectbindings"

	// UserProjectBindingKind represents "Kind" defined in Kubernetes
	UserProjectBindingKind = "UserProjectBinding"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// UserProjectBinding specifies a binding between a user and a project
// This resource is used by the user management to manipulate members of the given project
type UserProjectBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec UserProjectBindingSpec `json:"spec,omitempty"`
}

// UserProjectBindingSpec specifies a user
type UserProjectBindingSpec struct {
	UserEmail string `json:"userEmail"`
	ProjectID string `json:"projectID"`
	Group     string `json:"group"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// UserProjectBindingList is a list of users
type UserProjectBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []UserProjectBinding `json:"items"`
}
