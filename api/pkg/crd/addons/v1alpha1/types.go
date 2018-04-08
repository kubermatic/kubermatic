package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	AddonPlural = "addons"
)

//+genclient
//+genclient:nonNamespaced

// Addon specifies a add-on
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Addon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AddonSpec `json:"spec"`
}

type AddonMode string

const (
	AddonLastAppliedAnnotationKey = GroupName + "/last-applied-configuration"
	AddonModeAnnotationKey        = "addonmanager.kubernetes.io/mode"

	//AddonModeReconcile tells that the addon manager should overwrite any change made to the manifest
	AddonModeReconcile AddonMode = "Reconcile"
	//AddonModeEnsureExists tells that the addon manager should only make sure the manifest exist. The manifest can later get modified and wont be overwritten
	AddonModeEnsureExists AddonMode = "EnsureExists"
)

// AddonSpec specifies details of an addon
type AddonSpec struct {
	//Absolute path to the manifest templates
	Path string `json:"path"`
	//Free form data to use for parsing the manifest templates
	Variables runtime.RawExtension `json:"variables"`
}

// AddonList is a list of addons
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type AddonList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Addon `json:"items"`
}
