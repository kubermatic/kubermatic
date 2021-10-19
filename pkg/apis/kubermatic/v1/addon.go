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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// AddonResourceName represents "Resource" defined in Kubernetes
	AddonResourceName = "addons"

	// AddonKindName represents "Kind" defined in Kubernetes
	AddonKindName = "Addon"

	AddonResourcesCreated AddonConditionType = "AddonResourcesCreatedSuccessfully"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Addon specifies a add-on
type Addon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AddonSpec   `json:"spec"`
	Status AddonStatus `json:"status,omitempty"`
}

// GroupVersionKind unambiguously identifies a kind.  It doesn't anonymously include GroupVersion
// to avoid automatic coercion.  It doesn't use a GroupVersion to avoid custom marshalling
type GroupVersionKind struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

// AddonSpec specifies details of an addon
type AddonSpec struct {
	// Name defines the name of the addon to install
	Name string `json:"name"`
	// Cluster is the reference to the cluster the addon should be installed in
	Cluster corev1.ObjectReference `json:"cluster"`
	// Variables is free form data to use for parsing the manifest templates
	Variables runtime.RawExtension `json:"variables,omitempty"`
	// RequiredResourceTypes allows to indicate that this addon needs some resource type before it
	// can be installed. This can be used to indicate that a specific CRD and/or extension
	// apiserver must be installed before this addon can be installed. The addon will not
	// be installed until that resource is served.
	RequiredResourceTypes []GroupVersionKind `json:"requiredResourceTypes,omitempty"`
	// IsDefault indicates whether the addon is default
	IsDefault bool `json:"isDefault,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// AddonList is a list of addons
type AddonList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Addon `json:"items"`
}

type AddonStatus struct {
	Conditions []AddonCondition `json:"conditions,omitempty"`
}

type AddonConditionType string

type AddonCondition struct {
	// Type of addon condition.
	Type AddonConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
}
