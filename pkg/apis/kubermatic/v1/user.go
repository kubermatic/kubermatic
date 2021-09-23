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
	// UserResourceName represents "Resource" defined in Kubernetes
	UserResourceName = "users"

	// UserKindName represents "Kind" defined in Kubernetes
	UserKindName = "User"
)

//+genclient
//+genclient:nonNamespaced

// User specifies a user
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec UserSpec `json:"spec"`
}

// UserSpec specifies a user
type UserSpec struct {
	ID                      string                                  `json:"id"`
	Name                    string                                  `json:"name"`
	Email                   string                                  `json:"email"`
	IsAdmin                 bool                                    `json:"admin"`
	Settings                *UserSettings                           `json:"settings,omitempty"`
	TokenBlackListReference *providerconfig.GlobalSecretKeySelector `json:"tokenBlackListReference,omitempty"`
}

// UserSettings represent an user settings
type UserSettings struct {
	SelectedTheme              string `json:"selectedTheme,omitempty"`
	ItemsPerPage               int8   `json:"itemsPerPage,omitempty"`
	SelectedProjectID          string `json:"selectedProjectID,omitempty"`
	SelectProjectTableView     bool   `json:"selectProjectTableView,omitempty"`
	CollapseSidenav            bool   `json:"collapseSidenav,omitempty"`
	DisplayAllProjectsForAdmin bool   `json:"displayAllProjectsForAdmin,omitempty"`
	LastSeenChangelogVersion   string `json:"lastSeenChangelogVersion,omitempty"`
}

// UserList is a list of users
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []User `json:"items"`
}

// ProjectGroup is a helper data structure that
// stores the information about a project and a group that
// a user belongs to
type ProjectGroup struct {
	Name  string `json:"name"`
	Group string `json:"group"`
}

func (u *User) GetTokenBlackListSecretName() string {
	return fmt.Sprintf("token-blacklist-%s", u.Name)
}
