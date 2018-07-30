package v1

import (
	"fmt"

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
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Email    string         `json:"email"`
	Projects []ProjectGroup `json:"projects,omitempty"`
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

// GroupForProject returns a corresponding group name for the given project name
func (u *User) GroupForProject(projectName string) (string, error) {
	for _, pg := range u.Spec.Projects {
		if pg.Name == projectName {
			return pg.Group, nil
		}
	}
	return "", fmt.Errorf("The user doesn't belong to the given project = %s", projectName)
}
