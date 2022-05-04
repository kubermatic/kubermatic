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
	"encoding/json"

	semverlib "github.com/Masterminds/semver/v3"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ApplicationInstallation describes a single installation of an Application.
type ApplicationInstallation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationInstallationSpec   `json:"spec,omitempty"`
	Status ApplicationInstallationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationInstallationList is a list of ApplicationInstallations.
type ApplicationInstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ApplicationInstallation `json:"items"`
}

type ApplicationInstallationSpec struct {
	// Namespace describe the desired state of the namespace where application will be created.
	Namespace NamespaceSpec `json:"namespace"`

	// ApplicationRef is a reference to identify which Application should be deployed
	ApplicationRef ApplicationRef `json:"applicationRef"`

	// Values describe overrides for manifest-rendering
	Values json.RawMessage `json:"values,omitempty"`
	// As kubebuilder does not support interface{} as a type, deferring json decoding, seems to be our best option (see https://github.com/kubernetes-sigs/controller-tools/issues/294#issuecomment-518379253)
}

// NamespaceSpec describe the desired state of the namespace where application will be created.
type NamespaceSpec struct {
	// Name is the namespace to deploy the Application into.
	// Should be a valid lowercase RFC1123 domain name
	// +kubebuilder:validation:Pattern:=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Type=string
	Name string `json:"name"`

	// +kubebuilder:default:=true
	// Create defines whether the namespace should be created if it does not exist. Defaults to true
	Create bool `json:"create"`

	// Labels of the namespace
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations of the namespace
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ApplicationRef describes a KKP-wide, unique reference to an Application.
type ApplicationRef struct {
	// Name of the Application.
	// Should be a valid lowercase RFC1123 domain name
	// +kubebuilder:validation:Pattern:=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Type=string
	Name string `json:"name"`

	// +kubebuilder:validation:Pattern:=v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?
	// +kubebuilder:validation:Type=string

	// Version of the Application. Must be a valid SemVer version
	Version Version `json:"version"`
	// (pattern taken from masterminds/semver we use https://github.com/Masterminds/semver/blob/master/version.go#L42)
}

// +kubebuilder:validation:Type=string

// Version wraps semverlib.Version. It is needed because kubebuilder does not accept structs with non-tagged fields, even if they have custom marshallers
// With this the CRD resource will have Version as string but operator code can work directly with the semverlib.Version struct
// (taken from https://github.com/kubernetes-sigs/controller-tools/blob/master/pkg/crd/testdata/cronjob_types.go#L283)
type Version struct {
	semverlib.Version
}

// ApplicationInstallationStatus denotes status information about an ApplicationInstallation.
type ApplicationInstallationStatus struct {
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
	// Conditions contains conditions an installation is in, its primary use case is status signaling between controllers or between controllers and the API
	Conditions []ApplicationInstallationCondition `json:"conditions,omitempty"`

	// ApplicationVersion contains information installing / removing application
	ApplicationVersion *ApplicationVersion `json:"applicationVersion,omitempty"`
}

type ApplicationInstallationCondition struct {
	// Type of cluster condition.
	Type ApplicationInstallationConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// (brief) reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
}

// +kubebuilder:validation:Enum=ManifestsRetrieved;ManifestsApplied;Ready
type ApplicationInstallationConditionType string

const (
	// ManifestsRetrieved indicates all necessary manifests have been fetched from the external source.
	ManifestsRetrieved ApplicationInstallationConditionType = "ManifestsRetrieved"
	// ManifestsApplied indicates that all manifests have been applied in the target user-cluster.
	ManifestsApplied ApplicationInstallationConditionType = "ManifestsApplied"
	// Ready describes all components have been successfully rolled out and are ready.
	Ready ApplicationInstallationConditionType = "Ready"
)
