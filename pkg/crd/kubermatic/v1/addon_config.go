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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+genclient
//+genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddonConfig specifies addon configuration
type AddonConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec AddonConfigSpec `json:"spec"`
}

// AddonConfigSpec specifies configuration of addon
type AddonConfigSpec struct {
	// ShortDescription of the configured addon that contains more detailed information about the addon,
	// it will be displayed in the addon details view in the UI
	ShortDescription string `json:"shortDescription,omitempty"`
	// Description of the configured addon, it will be displayed in the addon overview in the UI
	Description string `json:"description,omitempty"`
	// Logo of the configured addon, encoded in base64
	Logo string `json:"logo,omitempty"`
	// LogoFormat contains logo format of the configured addon, i.e. svg+xml.
	LogoFormat string `json:"logoFormat,omitempty"`
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

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// AddonConfigList is a list of addon configs
type AddonConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []AddonConfig `json:"items"`
}
