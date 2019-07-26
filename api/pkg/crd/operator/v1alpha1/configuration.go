package v1alpha1

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ExposeStrategy is the strategy to expose the cluster with.
type ExposeStrategy string

const (
	// NodePortStrategy creates a NodePort with a "nodeport-proxy.k8s.io/expose": "true" annotation to expose
	// all clusters on one central Service of type LoadBalancer via the NodePort proxy.
	NodePortStrategy ExposeStrategy = "NodePort"
	// LoadBalancerStrategy creates a LoadBalancer service per cluster.
	LoadBalancerStrategy ExposeStrategy = "LoadBalancer"
)

// DockerImage describes a Docker image.
type DockerImage struct {
	Repository string            `json:"repository"`
	Tag        string            `json:"tag"`
	PullPolicy corev1.PullPolicy `json:"pullPolicy"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubermaticConfiguration is the configuration required for running Kubermatic.
type KubermaticConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec KubermaticConfigurationSpec `json:"spec"`
}

// KubermaticConfigurationSpec is the spec for a Kubermatic installation.
type KubermaticConfigurationSpec struct {
	// Namespace into which to install Kubermatic
	Namespace string `json:"namespace,omitempty"`
	// Domain is the base domain where the dashboard shall be available.
	Domain string `json:"domain"`
	// IsMaster controls whether the dashboard and API will be deployed.
	IsMaster bool `json:"isMaster,omitempty"`
	// Datacenters is a YAML string containing the available node datacenters. Note that
	// this is deprecated and you should use explicit Datacenter CRDs instead.
	Datacenters string `json:"datacenters,omitempty"`
	// Secrets is a list of predefined credentials, like Docker registry authentication.
	Secrets KubermaticSecretsConfiguration `json:"secrets,omitempty"`
	// Auth defines keys and URLs for Dex.
	Auth KubermaticAuthConfiguration `json:"auth"`
	// FeatureGates are used to optionally enable certain features.
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
	// UI configures the dashboard.
	UI KubermaticUIConfiguration `json:"ui,omitempty"`
	// API configures the frontend REST API used by the dashboard.
	API KubermaticAPIConfiguration `json:"api,omitempty"`
	// SeedController configures the controller-manager.
	SeedController KubermaticSeedControllerConfiguration `json:"seedController,omitempty"`
	// MasterController configures the controller-manager.
	MasterController KubermaticMasterControllerConfiguration `json:"masterController,omitempty"`
	// MasterFiles is a map of additional files to mount into each master component.
	MasterFiles map[string]string `json:"masterFiles,omitempty"`
	// ExposeStrategy is the strategy to expose the cluster with.
	// Note: The `seed_dns_overwrite` setting of the `datacenters.yaml` doesn't have any effect if this is set to LoadBalancerStrategy.
	ExposeStrategy ExposeStrategy `json:"exposeStrategy,omitempty"`
}

// EnabledFeatureGates returns a sorted list of feature names that are currently enabled.
func (spec *KubermaticConfigurationSpec) EnabledFeatureGates() []string {
	enabled := make([]string, 0)

	for name, flag := range spec.FeatureGates {
		if flag {
			enabled = append(enabled, name)
		}
	}

	sort.Strings(enabled)

	return enabled
}

// KubermaticSecretsConfiguration is a list of predefined credentials, like Docker registry authentication.
type KubermaticSecretsConfiguration struct {
	// ImagePullSecret is used to authenticate against Docker registries.
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
}

// KubermaticAuthConfiguration defines keys and URLs for Dex.
type KubermaticAuthConfiguration struct {
	ClientID                 string `json:"clientID,omitempty"`
	TokenIssuer              string `json:"tokenIssuer,omitempty"`
	IssuerRedirectURL        string `json:"issuerRedirectURL,omitempty"`
	IssuerClientID           string `json:"issuerClientID,omitempty"`
	IssuerClientSecret       string `json:"issuerClientSecret,omitempty"`
	IssuerCookieKey          string `json:"issuerCookieKey,omitempty"`
	CABundle                 string `json:"cABundle,omitempty"`
	ServiceAccountKey        string `json:"serviceAccountKey,omitempty"`
	SkipTokenIssuerTLSVerify bool   `json:"skipTokenIssuerTLSVerify,omitempty"`
}

// SimpleFeatureGate is a helper for features with no additional options.
type SimpleFeatureGate struct {
	// Enabled enables the feature.
	Enabled bool `json:"enabled,omitempty"`
}

// KubermaticAPIConfiguration configures the dashboard.
type KubermaticAPIConfiguration struct {
	// Image is the Docker image containing the Kubermatic REST API.
	Image DockerImage `json:"image,omitempty"`
}

// KubermaticUIConfiguration configures the dashboard.
type KubermaticUIConfiguration struct {
	// Image is the Docker image containing the Kubermatic dashboard.
	Image DockerImage `json:"image,omitempty"`
	// Config sets flags for various dashboard features.
	Config string `json:"config,omitempty"`
	// Presets is a YAML string containing pre-defined credentials for cloud providers.
	Presets string `json:"presets,omitempty"`
}

// KubermaticSeedControllerConfiguration configures the Kubermatic seed controller-manager.
type KubermaticSeedControllerConfiguration struct {
	// Image is the Docker image containing the Kubermatic controller-manager.
	Image DockerImage `json:"image,omitempty"`
	// Addons controls the optional additions installed into each user cluster.
	Addons KubermaticAddonsConfiguration `json:"addons,omitempty"`
	// NodePortRange is the port range for customer clusters - this must match the NodePort
	// range of the seed cluster.
	NodePortRange string `json:"nodePortRange,omitempty"`
	// OverwriteRegistry specifies a custom Docker registry which will be used for all images
	// (user cluster control plane + addons)
	OverwriteRegistry string `json:"overwriteRegistry,omitempty"`
	// BackupStoreContainer is the container used for shipping etcd snapshots to a backup location.
	BackupStoreContainer string `json:"backupStoreContainer,omitempty"`
	// BackupCleanupContainer is the container used for removing expired backups from the storage location.
	BackupCleanupContainer string `json:"backupCleanupContainer,omitempty"`
	// KubermaticImage can be used to overwrite the Docker image that is deployed inside user clusters.
	KubermaticImage string `json:"kubermaticImage,omitempty"`
}

// KubermaticAddonsConfiguration controls the optional additions installed into each user cluster.
type KubermaticAddonsConfiguration struct {
	// Kubernetes controls the addons for Kubernetes-based clusters.
	Kubernetes KubermaticAddonConfiguration `json:"kubernetes,omitempty"`
	// Openshift controls the addons for Openshift-based clusters.
	Openshift KubermaticAddonConfiguration `json:"openshift,omitempty"`
}

// KubermaticAddonConfiguration describes the addons for a given cluster runtime.
type KubermaticAddonConfiguration struct {
	// Default is the list of addons to be installed by default into each cluster.
	Default []string `json:"default,omitempty"`
	// Image is the Docker image containing the possible addon manifests.
	Image DockerImage `json:"image,omitempty"`
}

// KubermaticMasterControllerConfiguration configures the Kubermatic master controller-manager.
type KubermaticMasterControllerConfiguration struct {
	// Image is the Docker image containing the Kubermatic master controller-manager.
	Image DockerImage `json:"image,omitempty"`
	// ProjectsMigrator configures the migrator for user projects.
	ProjectsMigrator KubermaticProjectsMigratorConfiguration `json:"projectsMigrator,omitempty"`
}

// KubermaticProjectsMigratorConfiguration configures the Kubermatic master controller-manager.
type KubermaticProjectsMigratorConfiguration struct {
	// DryRun makes the migrator only log the actions it would take.
	DryRun bool `json:"dryRun,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubermaticConfigurationList is a collection of KubermaticConfigurations.
type KubermaticConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KubermaticConfiguration `json:"items"`
}
