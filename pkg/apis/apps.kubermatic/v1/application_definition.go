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
	// ApplicationInstallationResourceName represents "Resource" defined in Kubernetes.
	ApplicationDefinitionResourceName = "applicationdefinitions"

	// ApplicationDefinitionKindName represents "Kind" defined in Kubernetes.
	ApplicationDefinitionKindName = "ApplicationDefinitions"
)

type HelmCredentials struct {
	// Username holds the ref and key in the secret for the username credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to helm or git
	Username *corev1.SecretKeySelector `json:"username,omitempty"`

	// Password holds the ref and key in the secret for the Password credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to helm or git
	Password *corev1.SecretKeySelector `json:"password,omitempty"`

	// RegistryConfigFile holds the ref and key in the secret for the registry credential file. The value is dockercfg
	// file that follows the same format rules as ~/.docker/config.json
	// The The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to helm or git
	RegistryConfigFile *corev1.SecretKeySelector `json:"registryConfigFile,omitempty"`
}

type HelmSource struct {
	// URl of the helm repository.
	// It can be an HTTP(s) repository (e.g. https://localhost/myrepo) or on OCI repository (e.g. oci://localhost:5000/myrepo).
	// +kubebuilder:validation:Pattern="^(http|https|oci)://.+"
	URL string `json:"url"`

	// Name of the Chart.
	// +kubebuilder:validation:MinLength=1
	ChartName string `json:"chartName"`

	// Version of the Chart.
	// +kubebuilder:validation:MinLength=1
	ChartVersion string `json:"chartVersion"`

	// Credentials are optional and hold the ref to the secret with helm credentials.
	// Either username / Password or registryConfigFile can be defined.
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
	// Authentication method. Either password or token or ssh-key.
	// if method is password then username and password must be defined.
	// if method is token then token must be defined.
	// if method is ssh-key then ssh-key must be defined.
	Method GitAuthMethod `json:"method"`

	// Username holds the ref and key in the secret for the username credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to helm or git
	Username *corev1.SecretKeySelector `json:"username,omitempty"`

	// Password holds the ref and key in the secret for the Password credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to helm or git
	Password *corev1.SecretKeySelector `json:"password,omitempty"`

	// Token holds the ref and key in the secret for the token credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to helm or git
	Token *corev1.SecretKeySelector `json:"token,omitempty"`

	// SSHKey holds the ref and key in the secret for the SshKey credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to helm or git
	SSHKey *corev1.SecretKeySelector `json:"sshKey,omitempty"`
}

type GitReference struct {
	// Branch to checkout. Only the last commit of the branch will be checkout in order to reduce the amount of data to download.
	// +optional
	Branch string `json:"branch,omitempty"`

	// Commit SHA in a Branch to checkout.
	//
	// It must be used in conjunction with branch field.
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
	// URL to the repository. Can be HTTP(s) (e.g. https://example.com/myrepo) or SSH (e.g. git://example.com[:port]/path/to/repo.git/)
	// +kubebuilder:validation:MinLength=1
	Remote string `json:"remote"`

	// Git reference to checkout.
	// For large repositories, we recommend to either use Tag, Branch or Branch+Commit. This allows a shallow clone, which dramatically speeds up performance
	Ref GitReference `json:"ref"`

	// Path of the "source" in the repository. default is repository root
	Path string `json:"path,omitempty"`

	// Credentials are optional and holds the git credentials
	Credentials *GitCredentials `json:"credentials,omitempty"`
}

type ApplicationSource struct {
	// Install Application from a Helm repository
	Helm *HelmSource `json:"helm,omitempty"`

	// Install application from a Git repository
	Git *GitSource `json:"git,omitempty"`
}

const (
	HelmTemplateMethod TemplateMethod = "helm"
)

// +kubebuilder:validation:Enum=helm
type TemplateMethod string

type ApplicationTemplate struct {
	// Defined how the source of the application (e.g Helm chart) is retrieved.
	// Exactly one type of source must be defined.
	Source ApplicationSource `json:"source"`

	// DependencyCredentials holds the credentials that may be needed for templating the application.
	DependencyCredentials *DependencyCredentials `json:"templateCredentials,omitempty"`
}

type DependencyCredentials struct {
	// HelmCredentials holds the ref to the secret with helm credentials needed to build helm dependencies.
	// It is not required when using helm as a source, as dependencies are already prepackaged in this case.
	// It's either username / password or a registryConfigFile can be defined.
	HelmCredentials *HelmCredentials `json:"helmCredentials,omitempty"`
}

type ApplicationVersion struct {
	// +kubebuilder:validation:Pattern:=v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?
	// +kubebuilder:validation:Type=string

	// Version of the application (e.g. v1.2.3)
	Version string `json:"version"`
	// (pattern taken from masterminds/semver we use https://github.com/Masterminds/semver/blob/master/version.go#L42)

	// Template defines how application is installed (source provenance, Method...)
	Template ApplicationTemplate `json:"template"`
}

// ApplicationDefinitionSpec defines the desired state of ApplicationDefinition.
type ApplicationDefinitionSpec struct {
	// Description of the application. what is its purpose
	Description string `json:"description"`

	// Method used to install the application
	Method TemplateMethod `json:"method"`

	// DefaultValues describe overrides for manifest-rendering in UI when creating an application.
	// +kubebuilder:pruning:PreserveUnknownFields
	DefaultValues *runtime.RawExtension `json:"defaultValues,omitempty"`

	// DefaultDeployOptions holds the settings specific to the templating method used to deploy the application.
	// These settings can be overridden in applicationInstallation.
	DefaultDeployOptions *DeployOptions `json:"defaultDeployOptions,omitempty"`

	// Available version for this application
	Versions []ApplicationVersion `json:"versions"`
}

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
