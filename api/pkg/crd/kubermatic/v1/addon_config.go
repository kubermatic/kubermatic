package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// AddonConfigResourceName represents "Resource" defined in Kubernetes
	AddonConfigResourceName = "addonconfigs"

	// AddonKindName represents "Kind" defined in Kubernetes
	AddonConfigKindName = "AddonConfig"
)

//+genclient

// AddonConfig specifies addon configuration
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AddonConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AddonConfigSpec `json:"spec"`
}

// AddonConfigSpec specifies configuration of addon
type AddonConfigSpec struct {
	// Description of the configured addon
	Description string `json:"description"`
	// TODO: Fix the types of Logo and Variables.
	// Logo of the configured addon
	Logo string `json:"description"`
	// Variables that can be set for configured addon
	Variables runtime.RawExtension `json:"variables,omitempty"`
}

// AddonConfigList is a list of addon configs
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AddonConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AddonConfig `json:"items"`
}
