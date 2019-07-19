package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubermaticConfiguration is the configuration required for running
// a seed cluster with Kubermatic.
type KubermaticConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KubermaticConfigurationSpec `json:"spec"`
}

// KubermaticConfigurationSpec is the spec for a Kubermatic seed cluster.
type KubermaticConfigurationSpec struct {
	// Domain is the base domain where the dashboard shall be available.
	Domain string `json:"domain"`
	// IsMaster controls whether the dashboard and API will be deployed.
	IsMaster bool `json:"isMaster,omitempty"`
	// Secrets is a list of predefined credentials, like Docker registry authentication.
	Secrets KubermaticSecretsConfiguration `json:"secrets,omitempty"`
	// Auth defines keys and URLs for Dex.
	Auth KubermaticAuthConfiguration `json:"auth"`
	// FeatureGates are used to optionally enable certain features.
	FeatureGates KubermaticFeatureGatesConfiguration `json:"featureGates,omitempty"`
	// UI configures the dashboard.
	UI KubermaticUIConfiguration `json:"ui,omitempty"`
}

// KubermaticSecretsConfiguration is a list of predefined credentials, like Docker registry authentication.
type KubermaticSecretsConfiguration struct {
	// ImagePullSecret is used to authenticate against Docker registries.
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
}

// KubermaticAuthConfiguration defines keys and URLs for Dex.
type KubermaticAuthConfiguration struct {
	ClientID           string `json:"clientID,omitempty"`
	TokenIssuer        string `json:"tokenIssuer,omitempty"`
	IssuerRedirectURL  string `json:"issuerRedirectURL,omitempty"`
	IssuerClientID     string `json:"issuerClientID,omitempty"`
	IssuerClientSecret string `json:"issuerClientSecret,omitempty"`
	IssuerCookieKey    string `json:"issuerCookieKey,omitempty"`
	CABundle           string `json:"cABundle,omitempty"`
	ServiceAccountKey  string `json:"serviceAccountKey,omitempty"`
}

// KubermaticFeatureGatesConfiguration are used to optionally enable certain features.
type KubermaticFeatureGatesConfiguration struct {
	OIDCKubeCfgEndpoint   SimpleFeatureGate `json:"OIDCKubeCfgEndpoint,omitempty"`
	OpenIDAuthPlugin      SimpleFeatureGate `json:"OpenIDAuthPlugin,omitempty"`
	VerticalPodAutoscaler SimpleFeatureGate `json:"verticalPodAutoscaler,omitempty"`
}

// SimpleFeatureGate is a helper for features with no additional options.
type SimpleFeatureGate struct {
	// Enabled enables the feature.
	Enabled bool `json:"enabled,omitempty"`
}

// KubermaticUIConfiguration configures the dashboard.
type KubermaticUIConfiguration struct {
	// Config sets flags for various dashboard features.
	Config string `json:"config,omitempty"`
}
