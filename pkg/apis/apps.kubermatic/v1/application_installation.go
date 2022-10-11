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
	semverlib "github.com/Masterminds/semver/v3"
	"helm.sh/helm/v3/pkg/release"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// ApplicationInstallationResourceName represents "Resource" defined in Kubernetes.
	ApplicationInstallationResourceName = "applicationinstallations"

	// ApplicationInstallationKindName represents "Kind" defined in Kubernetes.
	ApplicationInstallationKindName = "ApplicationInstallations"

	// ApplicationInstallationsFQDNName represents "FQDN" defined in Kubernetes.
	ApplicationInstallationsFQDNName = ApplicationInstallationResourceName + "." + GroupName
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
	Namespace AppNamespaceSpec `json:"namespace"`

	// ApplicationRef is a reference to identify which Application should be deployed
	ApplicationRef ApplicationRef `json:"applicationRef"`

	// Values describe overrides for manifest-rendering. It's a free yaml field.
	// +kubebuilder:pruning:PreserveUnknownFields
	Values runtime.RawExtension `json:"values,omitempty"`
	// As kubebuilder does not support interface{} as a type, deferring json decoding, seems to be our best option (see https://github.com/kubernetes-sigs/controller-tools/issues/294#issuecomment-518379253)
}

// AppNamespaceSpec describe the desired state of the namespace where application will be created.
type AppNamespaceSpec struct {
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
// swagger:ignore
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
	// Conditions contains conditions an installation is in, its primary use case is status signaling between controllers or between controllers and the API
	Conditions map[ApplicationInstallationConditionType]ApplicationInstallationCondition `json:"conditions,omitempty"`

	// ApplicationVersion contains information installing / removing application
	ApplicationVersion *ApplicationVersion `json:"applicationVersion,omitempty"`

	// Method used to install the application
	Method TemplateMethod `json:"method"`

	// HelmRelease holds the information about the helm release installed by this application. This field is only filled if template method is 'helm'.
	HelmRelease *HelmRelease `json:"helmRelease,omitempty"`
}

type HelmRelease struct {
	// Name is the name of the release.
	Name string `json:"name,omitempty"`

	// Version is an int which represents the revision of the release.
	Version int `json:"version,omitempty"`

	// Info provides information about a release.
	Info *HelmReleaseInfo `json:"info,omitempty"`
}

// HelmReleaseInfo describes release information.
// tech note: we can not use release.Info from Helm because the underlying type used for time has no json tag.
type HelmReleaseInfo struct {
	// FirstDeployed is when the release was first deployed.
	FirstDeployed metav1.Time `json:"firstDeployed,omitempty"`

	// LastDeployed is when the release was last deployed.
	LastDeployed metav1.Time `json:"lastDeployed,omitempty"`

	// Deleted tracks when this object was deleted.
	Deleted metav1.Time `json:"deleted,omitempty"`

	// Description is human-friendly "log entry" about this release.
	Description string `json:"description,omitempty"`

	// Status is the current state of the release.
	Status release.Status `json:"status,omitempty"`

	// Notes is  the rendered templates/NOTES.txt if available.
	Notes string `json:"notes,omitempty"`
}

type ApplicationInstallationCondition struct {
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
}

// +kubebuilder:validation:Enum=ManifestsRetrieved;Ready

// swagger:enum ApplicationInstallationConditionType
// All condition types must be registered within the `AllApplicationInstallationConditionTypes` variable.
type ApplicationInstallationConditionType string

const (
	// ManifestsRetrieved indicates all necessary manifests have been fetched from the external source.
	ManifestsRetrieved ApplicationInstallationConditionType = "ManifestsRetrieved"

	// Ready describes all components have been successfully rolled out and are ready.
	Ready ApplicationInstallationConditionType = "Ready"
)

var AllApplicationInstallationConditionTypes = []ApplicationInstallationConditionType{
	ManifestsRetrieved,
	Ready,
}
