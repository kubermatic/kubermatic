package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+genclient
//+genclient:nonNamespaced

// UserProjectBinding specifies a binding between a user and a project
// This resource is used by the user management to manipulate members of the given project
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UserProjectBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec UserProjectBindingSpec `json:"spec"`
}

// UserProjectBindingSpec specifies a user
type UserProjectBindingSpec struct {
	UserEmail string `json:"userEmail"`
	ProjectID string `json:"projectId"`
	Group     string `json:"group"`
}

// UserList is a list of users
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UserProjectBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []UserProjectBinding `json:"items"`
}
