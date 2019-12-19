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
//+genclient:noStatus

// Addon specifies a add-on
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Addon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddonSpec   `json:"spec"`
	Status AddonStatus `json:"status"`
}

// AddonSpec specifies details of an addon
type AddonSpec struct {
	// Name defines the name of the addon to install
	Name string `json:"name"`
	// Cluster is the reference to the cluster the addon should be installed in
	Cluster corev1.ObjectReference `json:"cluster"`
	// Variables is free form data to use for parsing the manifest templates
	Variables runtime.RawExtension `json:"variables,omitempty"`
	// IsDefault indicates whether the addon is default
	IsDefault bool `json:"isDefault,omitempty"`
}

// AddonStatus contains details about the current status of an addon
type AddonStatus struct {
	// LastSuccessfulDeploymentTimestamp contains the time at which the addon
	// was successfully deployed last. It is nil if the addon was never deployed
	// successfully.
	LastSuccessfulDeploymentTimestamp *metav1.Time `json:"lastSuccessfulDeploymentTimestamp,omitempty"`
}

// AddonList is a list of addons
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AddonList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Addon `json:"items"`
}
