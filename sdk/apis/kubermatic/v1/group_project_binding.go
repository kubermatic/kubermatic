/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

	// GroupProjectBindingResourceName represents "Resource" defined in Kubernetes.
	GroupProjectBindingResourceName = "groupprojectbindings"

	// GroupProjectBindingKind represents "Kind" defined in Kubernetes.
	GroupProjectBindingKind = "GroupProjectBinding"

	// AuthZRoleLabel is the label used by rbac-controller and group-rbac-controller to identify the KKP role a ClusterRole or Role were created for.
	AuthZRoleLabel = "authz.k8c.io/role"

	// AuthZGroupProjectBindingLabel references the GroupProjectBinding resource that a ClusterRole/Role was created for.
	AuthZGroupProjectBindingLabel = "authz.k8c.io/group-project-binding"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".spec.projectID",name="ProjectID",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.group",name="Group",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.role",name="Role",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// GroupProjectBinding specifies a binding between a group and a project
// This resource is used by the user management to manipulate member groups of the given project.
type GroupProjectBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes an oidc group binding to a project.
	Spec GroupProjectBindingSpec `json:"spec,omitempty"`
}

// GroupProjectBindingSpec specifies an oidc group binding to a project.
type GroupProjectBindingSpec struct {
	// Group is the group name that is bound to the given project.
	Group string `json:"group"`
	// ProjectID is the ID of the target project.
	// Should be a valid lowercase RFC1123 domain name
	// +kubebuilder:validation:Pattern:=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Type=string
	ProjectID string `json:"projectID"`

	// +kubebuilder:validation:Enum=viewers;editors;owners;

	// Role is the user's role within the project, determining their permissions.
	// Possible roles are:
	// "viewers" - allowed to get/list project resources
	// "editors" - allowed to edit all project resources
	// "owners" - same as editors, but also can manage users in the project
	Role string `json:"role"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// GroupProjectBindingList is a list of group project bindings.
type GroupProjectBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items holds the list of the group and project bindings.
	Items []GroupProjectBinding `json:"items"`
}
