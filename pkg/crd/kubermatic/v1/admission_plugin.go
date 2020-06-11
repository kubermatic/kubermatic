package v1

import (
	"github.com/kubermatic/kubermatic/api/pkg/semver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AdmissionPluginList is the type representing a AdmissionPluginList
type AdmissionPluginList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of Admission Plugins
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md
	Items []AdmissionPlugin `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AdmissionPlugin is the type representing a AdmissionPlugin
type AdmissionPlugin struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AdmissionPluginSpec `json:"spec"`
}

// AdmissionPluginSpec specifies admission plugin name and from which k8s version is supported.
type AdmissionPluginSpec struct {
	PluginName string `json:"pluginName"`

	// FromVersion flag can be empty. It means the plugin fit to all k8s versions
	FromVersion *semver.Semver `json:"fromVersion,omitempty"`
}
