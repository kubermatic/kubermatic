package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// AddonConfigResourceName represents "Resource" defined in Kubernetes
	AddonConfigResourceName = "addonconfigs"

	// AddonKindName represents "Kind" defined in Kubernetes
	AddonConfigKindName = "AddonConfig"

	// ControlTypeBool represents bool type.
	ControlTypeBool = "bool"

	// ControlTypeString represents string type.
	ControlTypeString = "string"

	// ControlTypeNumber represents number type.
	ControlTypeNumber = "number"
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
	Description string `json:"description,omitempty"`
	// Logo of the configured addon, encoded in base64
	Logo string `json:"logo,omitempty"`
	// Controls that can be set for configured addon
	Controls []AddonFormControl `json:"formSpec,omitempty"`
}

// AddonFormControl specifies addon form control
type AddonFormControl struct {
	// DisplayName is visible in the UI
	DisplayName string `json:"displayName,omitempty"`
	// InternalName is used internally to save in the addon object
	InternalName string `json:"internalName,omitempty"`
	// Required indicates if the control has to be set
	Required bool `json:"required,omitempty"`
	// Type of displayed control
	Type string `json:"type,omitempty"`
}

// AddonConfigList is a list of addon configs
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AddonConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AddonConfig `json:"items"`
}
