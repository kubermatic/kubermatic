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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type HelmCredentials struct {
	// Username holds the ref and key in the secret for the username credential. Secret must exist in the namespace where
	// KKP is installed.
	Username corev1.SecretKeySelector `json:"username"`

	// Password holds the ref and key in the secret for the Password credential. Secret must exist in the namespace where
	// KKP is installed.
	Password corev1.SecretKeySelector `json:"password"`
}

type HelmSource struct {
	// URl of the helm repository.
	// It can be an HTTP(s) repository (e.g. https://localhost/myrepo) or on OCI repository (e.g. oci://localhost:5000/myrepo).
	// +kubebuilder:validation:Pattern="^(http|https|oci)://.+"
	URL string `json:"url"`

	// Name of the Chart
	// +kubebuilder:validation:MinLength=1
	ChartName string `json:"chartName"`

	// Version of the Chart
	// +kubebuilder:validation:MinLength=1
	ChartVersion string `json:"chartVersion"`

	// Credentials hold the ref to the secret with helm credentials
	Credentials *HelmCredentials `json:"credentials,omitempty"`
}

const (
	GitAuthMethodPassword GitAuthMethod = "password"
	GitAuthMethodToken    GitAuthMethod = "token"
	GitAuthMethodSSHKey   GitAuthMethod = "ssh-key"
)

// +kubebuilder:validation:Enum=password;token;ssh-key
type GitAuthMethod string

type GitCredentials struct {
	// Authentication method
	Method GitAuthMethod `json:"method"`

	// Username holds the ref and key in the secret for the username credential. Secret must exist in the namespace where
	// KKP is installed.
	Username *corev1.SecretKeySelector `json:"username,omitempty"`

	// Password holds the ref and key in the secret for the Password credential. Secret must exist in the namespace where
	// KKP is installed.
	Password *corev1.SecretKeySelector `json:"password,omitempty"`

	// Token holds the ref and key in the secret for the token credential. Secret must exist in the namespace where
	// KKP is installed.
	Token *corev1.SecretKeySelector `json:"token,omitempty"`

	// SSHKey holds the ref and key in the secret for the SshKey credential. Secret must exist in the namespace where
	// KKP is installed.
	SSHKey *corev1.SecretKeySelector `json:"sshKey,omitempty"`
}

type GitReference struct {
	// Branch to checkout. Only the last commit of the branch will be checkout in order to reduce the amount of data to download.
	// +optional
	Branch string `json:"branch,omitempty"`

	// Commit SHA to checkout.
	//
	// It can be used in conjunction with branch to to avoid to clone the all repository. The commit must belong to this branch.
	// +kubebuilder:validation:Pattern:=`^[a-f0-9]{40}$`
	// +kubebuilder:validation:Type=string
	// +optional
	Commit string `json:"commit,omitempty"`

	// Tag to check out.
	// It can not be used in conjunction with commit or branch.
	// +kubebuilder:validation:Type=string
	// +optional
	Tag string `json:"tag,omitempty"`
}

type GitSource struct {
	// URL to the repository (e.g. git://host.xz[:port]/path/to/repo.git/)
	// +kubebuilder:validation:MinLength=1
	Remote string `json:"remote"`

	// Git reference to checkout.
	//
	// For large repositories, we recommend to either use Tag, Branch or Branch+Commit. This allows a shallow clone, which dramatically speeds up performance
	Ref GitReference `json:"ref"`

	// Path of the "source" in the repository. default is repository root
	Path string `json:"path,omitempty"`

	// Credentials holds the git credentials
	Credentials *GitCredentials `json:"credentials,omitempty"`
}

type ApplicationSource struct {
	// Get application to install from a Helm repository
	Helm *HelmSource `json:"helm,omitempty"`

	// Get application to install from a Git repository
	Git *GitSource `json:"git,omitempty"`
}

const (
	FormFieldTypeNumber   FormFieldType = "number"
	FormFieldTypeBoolean  FormFieldType = "boolean"
	FormFieldTypeText     FormFieldType = "text"
	FormFieldTypeTextArea FormFieldType = "text-area"
)

// +kubebuilder:validation:Enum=number;boolean;text;text-area
type FormFieldType string

type FormField struct {
	// Type of displayed control
	Type FormFieldType `json:"type"`

	// DisplayName is visible in the UI
	DisplayName string `json:"displayName"`

	// InternalName is used internally to save in the ApplicationInstallation object
	InternalName string `json:"internalName"`

	// HelpText is visible in the UI next to the field
	HelpText string `json:"helpText,omitempty"`

	// Required indicates if the control has to be set
	Required bool `json:"required,omitempty"`
}

const (
	HelmTemplateMethod TemplateMethod = "helm"
)

// +kubebuilder:validation:Enum=helm
type TemplateMethod string

type ApplicationTemplate struct {
	// Defined how the source of the application (e.g Helm chart) is retrieved
	Source ApplicationSource `json:"source"`

	// Method used to install the application
	Method TemplateMethod `json:"method"`

	// Define the valued that can be override for the installation
	FormSpec []FormField `json:"formSpec,omitempty"`
}

type VersionConstraints string

func (c VersionConstraints) SemverConstraints() (*semverlib.Constraints, error) {
	return semverlib.NewConstraint(string(c))
}

type ApplicationConstraints struct {
	// Version of K8s version that a user cluster must satisfy for the application to be installed.
	// Wildcards are allowed  e.g. "1.18.*"
	// Range are allowed e.g." >= 1.19.0, < 1.19.15"
	K8sVersion VersionConstraints `json:"k8sVersion,omitempty"`

	// Version of KKP version that a user cluster must satisfy for the application to be installed.
	// Wildcards are allowed  e.g. "2.18.*"
	// Range are allowed e.g.">= 2.16, < 2.18.0"
	KKPVersion VersionConstraints `json:"kkpVersion,omitempty"`
}

type ApplicationVersion struct {
	// Version of the application (eg v1.2.3)
	Version string `json:"version"`

	// Constraints defined criteria that a user cluster must satisfy for the application to be installed
	Constraints ApplicationConstraints `json:"constraints,omitempty"`

	// Template defines how application is installed (source provenance, Method...)
	Template ApplicationTemplate `json:"template"`
}

// ApplicationDefinitionSpec defines the desired state of ApplicationDefinition.
type ApplicationDefinitionSpec struct {
	// Description of the application. what is its purpose
	Description string `json:"description"`

	// available version for this application
	Versions []ApplicationVersion `json:"versions"`
}

const (
	// ApplicationDefinitionResourceName represents "Resource" defined in Kubernetes.
	ApplicationDefinitionResourceName = "applicationdefinitions"
)

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster

// ApplicationDefinition is the Schema for the applicationdefinitions API.
type ApplicationDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ApplicationDefinitionSpec `json:"spec,omitempty"`
}

//+kubebuilder:object:root=true

// ApplicationDefinitionList contains a list of ApplicationDefinition.
type ApplicationDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ApplicationDefinition `json:"items"`
}
