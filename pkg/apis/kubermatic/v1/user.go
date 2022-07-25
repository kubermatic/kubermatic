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
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// UserResourceName represents "Resource" defined in Kubernetes.
	UserResourceName = "users"

	// UserKindName represents "Kind" defined in Kubernetes.
	UserKindName = "User"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.email",name="Email",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.name",name="HumanReadableName",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.admin",name="Admin",type="boolean"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// User specifies a user.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

// UserStatus stores status information about a user.
type UserStatus struct {
	// +optional
	LastSeen metav1.Time `json:"lastSeen,omitempty"`
}

// UserSpec specifies a user.
type UserSpec struct {
	// ID is an unnused legacy field.
	// Deprecated: do not set this field anymore.
	ID string `json:"id,omitempty"`
	// Name is the full name of this user.
	Name string `json:"name"`
	// Email is the email address of this user. Emails must be globally unique across
	// all KKP users.
	Email string `json:"email"`
	// IsAdmin defines whether this user is an administrator with additional permissions.
	// Admins can for example see all projects and clusters in the KKP dashboard.
	// +kubebuilder:default=false
	IsAdmin bool `json:"admin"`
	// Groups holds the information to which groups the user belongs to. Set automatically when logging in to the
	// KKP API, and used by the KKP API.
	Groups []string `json:"groups,omitempty"`

	// Project is the name of the project that this service account user is tied to. This
	// field is only applicable to service accounts and regular users must not set this field.
	// +optional
	Project string `json:"project,omitempty"`

	Settings               *UserSettings                           `json:"settings,omitempty"`
	InvalidTokensReference *providerconfig.GlobalSecretKeySelector `json:"invalidTokensReference,omitempty"`
}

// UserSettings represent an user settings.
type UserSettings struct {
	SelectedTheme              string `json:"selectedTheme,omitempty"`
	ItemsPerPage               int8   `json:"itemsPerPage,omitempty"`
	SelectedProjectID          string `json:"selectedProjectID,omitempty"`
	SelectProjectTableView     bool   `json:"selectProjectTableView,omitempty"`
	CollapseSidenav            bool   `json:"collapseSidenav,omitempty"`
	DisplayAllProjectsForAdmin bool   `json:"displayAllProjectsForAdmin,omitempty"`
	LastSeenChangelogVersion   string `json:"lastSeenChangelogVersion,omitempty"`
	UseClustersView            bool   `json:"useClustersView,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// UserList is a list of users.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []User `json:"items"`
}

// ProjectGroup is a helper data structure that
// stores the information about a project and a group that
// a user belongs to.
type ProjectGroup struct {
	Name  string `json:"name"`
	Group string `json:"group"`
}

func (u *User) GetInvalidTokensReferenceSecretName() string {
	// "token-blacklist-" is the legacy prefix; changing this would mean existing
	// secrets would need to be migrated first
	return fmt.Sprintf("token-blacklist-%s", u.Name)
}
