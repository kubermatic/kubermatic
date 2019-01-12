package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// AddonResourceName represents "Resource" defined in Kubernetes
	AddonResourceName = "addons"

	// AddonKindName represents "Kind" defined in Kubernetes
	AddonKindName = "Addon"
)

//+genclient

// Addon specifies a add-on
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Addon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AddonSpec `json:"spec"`
}

// AddonSpec specifies details of an addon
type AddonSpec struct {
	// Name defines the name of the addon to install
	Name string `json:"name"`
	// Cluster is the reference to the cluster the addon should be installed in
	Cluster corev1.ObjectReference `json:"cluster"`
	// Variables is free form data to use for parsing the manifest templates
	Variables runtime.RawExtension `json:"variables,omitempty"`
}

// AddonList is a list of addons
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AddonList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Addon `json:"items"`
}
