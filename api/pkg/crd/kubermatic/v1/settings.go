package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Settings is the type representing a Settings
type Settings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SettingsSpec `json:"spec"`
}

type SettingsSpec struct {
	CustomLinks           CustomLinks    `json:"customLinks"`
	CleanupOptions        CleanupOptions `json:"cleanupOptions"`
	DefaultNodeCount      int8           `json:"defaultNodeCount"`
	ClusterTypeOptions    int8           `json:"clusterTypeOptions"`
	DisplayDemoInfo       bool           `json:"displayDemoInfo"`
	DisplayAPIDocs        bool           `json:"displayAPIDocs"`
	DisplayTermsOfService bool           `json:"displayTermsOfService"`

	// TODO: Datacenters, presets, user management, Google Analytics and default addons.
}

type CustomLinks []CustomLink

type CustomLink struct {
	Label    string `json:"label"`
	URL      string `json:"url"`
	Icon     string `json:"icon"`
	Location string `json:"location"`
}

type CleanupOptions struct {
	Enabled  bool
	Enforced bool
}
