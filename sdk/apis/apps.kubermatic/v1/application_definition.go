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
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

const (
	// ApplicationDefinitionResourceName represents "Resource" defined in Kubernetes.
	ApplicationDefinitionResourceName = "applicationdefinitions"

	// ApplicationDefinitionKindName represents "Kind" defined in Kubernetes.
	ApplicationDefinitionKindName = "ApplicationDefinitions"
)

type HelmCredentials struct {
	// Username holds the ref and key in the secret for the username credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to "helm" or "git"
	Username *corev1.SecretKeySelector `json:"username,omitempty"`

	// Password holds the ref and key in the secret for the password credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to "helm" or "git"
	Password *corev1.SecretKeySelector `json:"password,omitempty"`

	// RegistryConfigFile holds the ref and key in the secret for the registry credential file.
	// The value is dockercfg file that follows the same format rules as ~/.docker/config.json.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to "helm" or "git"
	RegistryConfigFile *corev1.SecretKeySelector `json:"registryConfigFile,omitempty"`
}

type HelmSource struct {
	// +kubebuilder:validation:Pattern="^(http|https|oci)://.+"

	// URL of the Helm repository the following schemes are supported:
	//
	// * http://example.com/myrepo (HTTP)
	// * https://example.com/myrepo (HTTPS)
	// * oci://example.com:5000/myrepo (OCI, HTTPS by default, use plainHTTP to enable unencrypted HTTP)
	URL string `json:"url"`

	// Insecure disables certificate validation when using an HTTPS registry. This setting has no
	// effect when using a plaintext connection.
	Insecure *bool `json:"insecure,omitempty"`

	// PlainHTTP will enable HTTP-only (i.e. unencrypted) traffic for oci:// URLs. By default HTTPS
	// is used when communicating with an oci:// URL.
	PlainHTTP *bool `json:"plainHTTP,omitempty"`

	// Name of the Chart.
	// +kubebuilder:validation:MinLength=1
	ChartName string `json:"chartName"`

	// Version of the Chart.
	// +kubebuilder:validation:MinLength=1
	ChartVersion string `json:"chartVersion"`

	// Credentials are optional and hold the ref to the secret with Helm credentials.
	// Either username / password or registryConfigFile can be defined.
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
	// If method is password then username and password must be defined.
	// If method is token then token must be defined.
	// If method is ssh-key then ssh-key must be defined.
	Method GitAuthMethod `json:"method"`

	// Username holds the ref and key in the secret for the username credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to "helm" or "git".
	Username *corev1.SecretKeySelector `json:"username,omitempty"`

	// Password holds the ref and key in the secret for the Password credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to "helm" or "git".
	Password *corev1.SecretKeySelector `json:"password,omitempty"`

	// Token holds the ref and key in the secret for the token credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to "helm" or "git".
	Token *corev1.SecretKeySelector `json:"token,omitempty"`

	// SSHKey holds the ref and key in the secret for the SshKey credential.
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to "helm" or "git".
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
	// URL to the repository. Can be HTTP(s) (e.g. https://example.com/myrepo) or
	// SSH (e.g. git://example.com[:port]/path/to/repo.git/).
	// +kubebuilder:validation:MinLength=1
	Remote string `json:"remote"`

	// Git reference to checkout.
	// For large repositories, we recommend to either use Tag, Branch or Branch+Commit.
	// This allows a shallow clone, which dramatically speeds up performance
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
	// DisplayName is the name for the application that will be displayed in the UI.
	DisplayName string `json:"displayName,omitempty"`

	// Description of the application. what is its purpose
	Description string `json:"description"`

	// Method used to install the application
	Method TemplateMethod `json:"method"`

	// DefaultValues specify default values for the UI which are passed to helm templating when creating an application. Comments are not preserved.
	// Deprecated: Use DefaultValuesBlock instead. This field was deprecated in KKP 2.25 and will be removed in KKP 2.27+.
	// +kubebuilder:pruning:PreserveUnknownFields
	DefaultValues *runtime.RawExtension `json:"defaultValues,omitempty"`

	// DefaultValuesBlock specifies default values for the UI which are passed to helm templating when creating an application. Comments are preserved.
	DefaultValuesBlock string `json:"defaultValuesBlock,omitempty"`

	// DefaultNamespace specifies the default namespace which is used if a referencing ApplicationInstallation has no target namespace defined.
	// If unset, the name of the ApplicationDefinition is being used instead.
	DefaultNamespace *AppNamespaceSpec `json:"defaultNamespace,omitempty"`

	// DefaultDeployOptions holds the settings specific to the templating method used to deploy the application.
	// These settings can be overridden in applicationInstallation.
	DefaultDeployOptions *DeployOptions `json:"defaultDeployOptions,omitempty"`

	// +kubebuilder:validation:Pattern:=v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?
	// +kubebuilder:validation:Type=string

	// DefaultVersion of the application to use, if not specified the latest available version will be used.
	// +optional
	DefaultVersion string `json:"defaultVersion,omitempty"`

	// Enforced specifies if the application is enforced to be installed on the user clusters. Enforced applications are
	// installed/updated by KKP for the user clusters. Users are not allowed to update/delete them. KKP will revert the changes
	// done by the application to the desired state specified in the ApplicationDefinition.
	// +optional
	Enforced bool `json:"enforced,omitempty"`

	// Default specifies if the application should be installed by default when a new user cluster is created. Default applications are
	// not enforced and users can update/delete them. KKP will only install them during cluster creation if the user didn't explicitly
	// opt out from installing default applications.
	// +optional
	Default bool `json:"default,omitempty"`

	// Selector is used to select the targeted user clusters for defaulting and enforcing applications. This is only used for default/enforced applications and ignored otherwise.
	// +optional
	Selector DefaultingSelector `json:"selector,omitempty"`

	// DocumentationURL holds a link to official documentation of the Application
	// Alternatively this can be a link to the Readme of a chart in a git repository
	DocumentationURL string `json:"documentationURL,omitempty"`

	// SourceURL holds a link to the official source code mirror or git repository of the application
	SourceURL string `json:"sourceURL,omitempty"`

	// Logo of the Application as a base64 encoded svg
	Logo string `json:"logo,omitempty"`

	// LogoFormat contains logo format of the configured Application. Options are "svg+xml" and "png"
	// +kubebuilder:validation:Enum=svg+xml;png
	LogoFormat string `json:"logoFormat,omitempty"`

	// Available version for this application
	Versions []ApplicationVersion `json:"versions"`
}

// DefaultingSelector is used to select the targeted user clusters for defaulting and enforcing applications.
type DefaultingSelector struct {
	// Datacenters is a list of datacenters where the application can be installed.
	Datacenters []string `json:"datacenters,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:resource:scope=Cluster,shortName=appdef

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

// GetDefaultValues parses the values either from the Values or ValuesBlock field.
// Will return an error if both fields are set.
// Will return nil if none of the fields are set.
func (ad *ApplicationDefinitionSpec) GetDefaultValues() ([]byte, error) {
	if ad.DefaultValues != nil && len(ad.DefaultValues.Raw) > 0 && ad.DefaultValuesBlock != "" {
		return nil, errors.New("the fields DefaultValues and DefaultValuesBlock cannot be used simultaneously, please delete one of them")
	}
	if ad.DefaultValues != nil && len(ad.DefaultValues.Raw) > 0 {
		return ad.DefaultValues.Raw, nil
	}
	if ad.DefaultValuesBlock != "" {
		return []byte(ad.DefaultValuesBlock), nil
	}
	return nil, nil
}

// GetParsedDefaultValues parses the values either from the DefaultValues or DefaultValuesBlock field.
// Will return an error if both fields are set.
func (ad *ApplicationDefinitionSpec) GetParsedDefaultValues() (map[string]interface{}, error) {
	values := make(map[string]interface{})
	if !isEmptyRawExtension(ad.DefaultValues) && ad.DefaultValuesBlock != "" {
		return nil, errors.New("the fields DefaultValues and DefaultValuesBlock cannot be used simultaneously, please delete one of them")
	}
	if !isEmptyRawExtension(ad.DefaultValues) {
		err := json.Unmarshal(ad.DefaultValues.Raw, &values)
		return values, err
	}
	err := yaml.Unmarshal([]byte(ad.DefaultValuesBlock), &values)
	return values, err
}
