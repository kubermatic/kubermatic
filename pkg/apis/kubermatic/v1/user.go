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
// +kubebuilder:printcolumn:JSONPath=".spec.name",name="Name",type="string"
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
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsAdmin bool   `json:"admin"`

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
