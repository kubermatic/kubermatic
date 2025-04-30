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
	"k8c.io/kubermatic/sdk/v2/semver"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=always;externalCloudProvider;inTreeProvider

// ConditionType is the type defining the cluster or datacenter condition that must be met to block a specific version.
type ConditionType string

const (
	// AlwaysCondition represent an always true matching condition used while checking provider incompatibilities.
	AlwaysCondition ConditionType = "always"
	// ExternalCloudProviderCondition is an incompatibility condition that represents the usage of the external Cloud Provider.
	ExternalCloudProviderCondition ConditionType = ClusterFeatureExternalCloudProvider
	// InTreeCloudProviderCondition is an incompatibility condition that represents the usage of the in-tree Cloud Provider.
	InTreeCloudProviderCondition ConditionType = "inTreeProvider"
)

// +kubebuilder:validation:Enum=CREATE;UPGRADE;SUPPORT

// OperationType is the type defining the operations triggering the compatibility check (CREATE or UPDATE).
type OperationType string

// +kubebuilder:validation:Enum=Addon;Cluster;ClusterBackup;Project;None
// MetricsCollector is the name of an available metrics collector.
type MetricsCollector string

const (
	// AddonCollector is addon metrics collector.
	AddonCollector MetricsCollector = "Addon"
	// ClusterBackupCollector is cluster backup metrics collector.
	ClusterBackupCollector MetricsCollector = "ClusterBackup"
	// ClusterCollector is cluster metrics collector.
	ClusterCollector MetricsCollector = "Cluster"
	// ProjectCollector is project metrics collector.
	ProjectCollector MetricsCollector = "Project"
	// NoneCollector is a special name that points to no collector.
	// It is used when a client wants to disable certain collectors in all seeds except for a few.
	NoneCollector MetricsCollector = "None"
)

const (
	// CreateOperation represents the creation of a new cluster.
	CreateOperation OperationType = "CREATE"
	// UpdateOperation represents the update of an existing cluster.
	UpdateOperation OperationType = "UPGRADE"
	// SupportOperation represents the possibility to enable a new feature on an existing cluster.
	SupportOperation OperationType = "SUPPORT"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// KubermaticConfiguration is the configuration required for running Kubermatic.
type KubermaticConfiguration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubermaticConfigurationSpec   `json:"spec,omitempty"`
	Status KubermaticConfigurationStatus `json:"status,omitempty"`
}

// KubermaticConfigurationStatus stores status information about a KubermaticConfiguration.
type KubermaticConfigurationStatus struct {
	// KubermaticVersion current Kubermatic Version.
	KubermaticVersion string `json:"kubermaticVersion,omitempty"`
	// KubermaticEdition current Kubermatic Edition , i.e. Community Edition or Enterprise Edition.
	KubermaticEdition string `json:"kubermaticEdition,omitempty"`
}

// KubermaticConfigurationSpec is the spec for a Kubermatic installation.
type KubermaticConfigurationSpec struct {
	// CABundle references a ConfigMap in the same namespace as the KubermaticConfiguration.
	// This ConfigMap must contain a ca-bundle.pem with PEM-encoded certificates. This bundle
	// automatically synchronized into each seed and each usercluster. APIGroup and Kind are
	// currently ignored.
	CABundle corev1.TypedLocalObjectReference `json:"caBundle,omitempty"`
	// ImagePullSecret is used to authenticate against Docker registries.
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
	// Auth defines keys and URLs for Dex. These must be defined unless the HeadlessInstallation
	// feature gate is set, which will disable the UI/API and its need for an OIDC provider entirely.
	// +optional
	Auth KubermaticAuthConfiguration `json:"auth"`
	// FeatureGates are used to optionally enable certain features.
	FeatureGates map[string]bool `json:"featureGates,omitempty"`
	// UI configures the dashboard.
	UI KubermaticUIConfiguration `json:"ui,omitempty"`
	// API configures the frontend REST API used by the dashboard.
	API KubermaticAPIConfiguration `json:"api,omitempty"`
	// SeedController configures the seed-controller-manager.
	SeedController KubermaticSeedControllerConfiguration `json:"seedController,omitempty"`
	// MasterController configures the master-controller-manager.
	MasterController KubermaticMasterControllerConfiguration `json:"masterController,omitempty"`
	// Webhook configures the webhook.
	Webhook KubermaticWebhookConfiguration `json:"webhook,omitempty"`
	// UserCluster configures various aspects of the user-created clusters.
	UserCluster KubermaticUserClusterConfiguration `json:"userCluster,omitempty"`
	// ExposeStrategy is the strategy to expose the cluster with.
	// Note: The `seed_dns_overwrite` setting of a Seed's datacenter doesn't have any effect
	// if this is set to LoadBalancerStrategy.
	ExposeStrategy ExposeStrategy `json:"exposeStrategy,omitempty"`
	// Ingress contains settings for making the API and UI accessible remotely.
	Ingress KubermaticIngressConfiguration `json:"ingress,omitempty"`
	// Versions configures the available and default Kubernetes versions and updates.
	Versions KubermaticVersioningConfiguration `json:"versions,omitempty"`
	// VerticalPodAutoscaler configures the Kubernetes VPA integration.
	VerticalPodAutoscaler KubermaticVPAConfiguration `json:"verticalPodAutoscaler,omitempty"`
	// Proxy allows to configure Kubermatic to use proxies to talk to the
	// world outside of its cluster.
	Proxy KubermaticProxyConfiguration `json:"proxy,omitempty"`
	// MirrorImages is a list of container images that will be mirrored with the `kubermatic-installer  mirror-images` command.
	// Each entry should be in the format "repository:tag".
	MirrorImages []string `json:"mirrorImages,omitempty"`
}

// KubermaticAuthConfiguration defines keys and URLs for Dex.
type KubermaticAuthConfiguration struct {
	ClientID                 string `json:"clientID,omitempty"`
	TokenIssuer              string `json:"tokenIssuer,omitempty"`
	IssuerRedirectURL        string `json:"issuerRedirectURL,omitempty"`
	IssuerClientID           string `json:"issuerClientID,omitempty"`
	IssuerClientSecret       string `json:"issuerClientSecret,omitempty"`
	IssuerCookieKey          string `json:"issuerCookieKey,omitempty"`
	ServiceAccountKey        string `json:"serviceAccountKey,omitempty"`
	SkipTokenIssuerTLSVerify bool   `json:"skipTokenIssuerTLSVerify,omitempty"`
}

// KubermaticAPIConfiguration configures the dashboard.
type KubermaticAPIConfiguration struct {
	// DockerRepository is the repository containing the Kubermatic REST API image.
	DockerRepository string `json:"dockerRepository,omitempty"`
	// DockerTag is used to overwrite the Kubermatic API Docker image tag and is only for development
	// purposes. This field must not be set in production environments. If DockerTag is specified then
	// DockerTagSuffix will be ignored.
	// ---
	//nolint:staticcheck
	//lint:ignore SA5008 omitgenyaml is used by the example-yaml-generator
	DockerTag string `json:"dockerTag,omitempty,omitgenyaml"`
	// DockerTagSuffix is appended to the KKP version used for referring to the custom Kubermatic API image.
	// If left empty, either the `DockerTag` if specified or the original Kubermatic API Docker image tag will be used.
	// With DockerTagSuffix the tag becomes <KKP_VERSION-SUFFIX> i.e. "v2.15.0-SUFFIX".
	DockerTagSuffix string `json:"dockerTagSuffix,omitempty"`
	// AccessibleAddons is a list of addons that should be enabled in the API.
	AccessibleAddons []string `json:"accessibleAddons,omitempty"`
	// PProfEndpoint controls the port the API should listen on to provide pprof
	// data. This port is never exposed from the container and only available via port-forwardings.
	PProfEndpoint *string `json:"pprofEndpoint,omitempty"`
	// Resources describes the requested and maximum allowed CPU/memory usage.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// DebugLog enables more verbose logging.
	DebugLog bool `json:"debugLog,omitempty"`
	// Replicas sets the number of pod replicas for the API deployment.
	Replicas *int32 `json:"replicas,omitempty"`
}

// KubermaticUIConfiguration configures the dashboard.
type KubermaticUIConfiguration struct {
	// DockerRepository is the repository containing the Kubermatic dashboard image.
	DockerRepository string `json:"dockerRepository,omitempty"`
	// DockerTag is used to overwrite the dashboard Docker image tag and is only for development
	// purposes. This field must not be set in production environments. If DockerTag is specified then
	// DockerTagSuffix will be ignored.
	// ---
	//nolint:staticcheck
	//lint:ignore SA5008 omitgenyaml is used by the example-yaml-generator
	DockerTag string `json:"dockerTag,omitempty,omitgenyaml"`
	// DockerTagSuffix is appended to the KKP version used for referring to the custom dashboard image.
	// If left empty, either the `DockerTag` if specified or the original dashboard Docker image tag will be used.
	// With DockerTagSuffix the tag becomes <KKP_VERSION-SUFFIX> i.e. "v2.15.0-SUFFIX".
	DockerTagSuffix string `json:"dockerTagSuffix,omitempty"`
	// Config sets flags for various dashboard features.
	Config string `json:"config,omitempty"`
	// Resources describes the requested and maximum allowed CPU/memory usage.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// Replicas sets the number of pod replicas for the UI deployment.
	Replicas *int32 `json:"replicas,omitempty"`
	// ExtraVolumeMounts allows to mount additional volumes into the UI container.
	ExtraVolumeMounts []corev1.VolumeMount `json:"extraVolumeMounts,omitempty"`
	// ExtraVolumes allows to mount additional volumes into the UI container.
	ExtraVolumes []corev1.Volume `json:"extraVolumes,omitempty"`
}

// KubermaticSeedControllerConfiguration configures the Kubermatic seed controller-manager.
type KubermaticSeedControllerConfiguration struct {
	// DockerRepository is the repository containing the Kubermatic seed-controller-manager image.
	DockerRepository string `json:"dockerRepository,omitempty"`
	// BackupStoreContainer is the container used for shipping etcd snapshots to a backup location.
	BackupStoreContainer string `json:"backupStoreContainer,omitempty"`
	// BackupDeleteContainer is the container used for deleting etcd snapshots from a backup location.
	BackupDeleteContainer string `json:"backupDeleteContainer,omitempty"`
	// Deprecated: BackupCleanupContainer is the container used for removing expired backups from the storage location.
	// This field is a no-op and is no longer used. The old backup controller it was used for has been
	// removed. Do not set this field.
	BackupCleanupContainer string `json:"backupCleanupContainer,omitempty"`
	// MaximumParallelReconciles limits the number of cluster reconciliations
	// that are active at any given time.
	MaximumParallelReconciles int `json:"maximumParallelReconciles,omitempty"`
	// PProfEndpoint controls the port the seed-controller-manager should listen on to provide pprof
	// data. This port is never exposed from the container and only available via port-forwardings.
	PProfEndpoint *string `json:"pprofEndpoint,omitempty"`
	// Resources describes the requested and maximum allowed CPU/memory usage.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// DebugLog enables more verbose logging.
	DebugLog bool `json:"debugLog,omitempty"`
	// Replicas sets the number of pod replicas for the seed-controller-manager.
	Replicas *int32 `json:"replicas,omitempty"`
	// DisabledCollectors contains a list of metrics collectors that should be disabled.
	// Acceptable values are "Addon", "Cluster", "ClusterBackup", "Project", and "None".
	DisabledCollectors []MetricsCollector `json:"disabledCollectors,omitempty"`
}

// KubermaticWebhookConfiguration configures the Kubermatic webhook.
type KubermaticWebhookConfiguration struct {
	// DockerRepository is the repository containing the Kubermatic webhook image.
	DockerRepository string `json:"dockerRepository,omitempty"`
	// PProfEndpoint controls the port the webhook should listen on to provide pprof
	// data. This port is never exposed from the container and only available via port-forwardings.
	PProfEndpoint *string `json:"pprofEndpoint,omitempty"`
	// Resources describes the requested and maximum allowed CPU/memory usage.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// DebugLog enables more verbose logging.
	DebugLog bool `json:"debugLog,omitempty"`
	// Replicas sets the number of pod replicas for the webhook.
	Replicas *int32 `json:"replicas,omitempty"`
}

// KubermaticUserClusterConfiguration controls various aspects of the user-created clusters.
type KubermaticUserClusterConfiguration struct {
	// KubermaticDockerRepository is the repository containing the Kubermatic user-cluster-controller-manager image.
	KubermaticDockerRepository string `json:"kubermaticDockerRepository,omitempty"`
	// DNATControllerDockerRepository is the repository containing the
	// dnat-controller image.
	DNATControllerDockerRepository string `json:"dnatControllerDockerRepository,omitempty"`
	// EtcdLauncherDockerRepository is the repository containing the Kubermatic
	// etcd-launcher image.
	EtcdLauncherDockerRepository string `json:"etcdLauncherDockerRepository,omitempty"`
	// OverwriteRegistry specifies a custom Docker registry which will be used for all images
	// used for user clusters (user cluster control plane + addons). This also applies to
	// the KubermaticDockerRepository and DNATControllerDockerRepository fields.
	OverwriteRegistry string `json:"overwriteRegistry,omitempty"`
	// Addons controls the optional additions installed into each user cluster.
	Addons KubermaticAddonsConfiguration `json:"addons,omitempty"`
	// SystemApplications contains configuration for system Applications (such as CNI).
	SystemApplications SystemApplicationsConfiguration `json:"systemApplications,omitempty"`
	// Applications contains configuration for default Application settings.
	Applications ApplicationsConfiguration `json:"applications,omitempty"`
	// NodePortRange is the port range for user clusters - this must match the NodePort
	// range of the seed cluster.
	NodePortRange string `json:"nodePortRange,omitempty"`
	// Monitoring can be used to fine-tune to in-cluster Prometheus.
	Monitoring KubermaticUserClusterMonitoringConfiguration `json:"monitoring,omitempty"`
	// DisableAPIServerEndpointReconciling can be used to toggle the `--endpoint-reconciler-type` flag for
	// the Kubernetes API server.
	DisableAPIServerEndpointReconciling bool `json:"disableApiserverEndpointReconciling,omitempty"`
	// EtcdVolumeSize configures the volume size to use for each etcd pod inside user clusters.
	EtcdVolumeSize string `json:"etcdVolumeSize,omitempty"`
	// APIServerReplicas configures the replica count for the API-Server deployment inside user clusters.
	APIServerReplicas *int32 `json:"apiserverReplicas,omitempty"`
	// MachineController configures the Machine Controller
	MachineController MachineControllerConfiguration `json:"machineController,omitempty"`
	// OperatingSystemManager configures the image repo and the tag version for osm deployment.
	OperatingSystemManager OperatingSystemManager `json:"operatingSystemManager,omitempty"`
}

// KubermaticUserClusterMonitoringConfiguration can be used to fine-tune to in-cluster Prometheus.
type KubermaticUserClusterMonitoringConfiguration struct {
	// DisableDefaultRules disables the recording and alerting rules.
	DisableDefaultRules bool `json:"disableDefaultRules,omitempty"`
	// DisableDefaultScrapingConfigs disables the default scraping targets.
	DisableDefaultScrapingConfigs bool `json:"disableDefaultScrapingConfigs,omitempty"`
	// CustomRules can be used to inject custom recording and alerting rules. This field
	// must be a YAML-formatted string with a `group` element at its root, as documented
	// on https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/.
	// This value is treated as a Go template, which allows to inject dynamic values like
	// the internal cluster address or the cluster ID. Refer to pkg/resources/prometheus
	// and the documentation for more information on the available fields.
	CustomRules string `json:"customRules,omitempty"`
	// CustomScrapingConfigs can be used to inject custom scraping rules. This must be a
	// YAML-formatted string containing an array of scrape configurations as documented
	// on https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config.
	// This value is treated as a Go template, which allows to inject dynamic values like
	// the internal cluster address or the cluster ID. Refer to pkg/resources/prometheus
	// and the documentation for more information on the available fields.
	CustomScrapingConfigs string `json:"customScrapingConfigs,omitempty"`
	// ScrapeAnnotationPrefix (if set) is used to make the in-cluster Prometheus scrape pods
	// inside the user clusters.
	ScrapeAnnotationPrefix string `json:"scrapeAnnotationPrefix,omitempty"`
}

// MachineControllerConfiguration configures Machine Controller.
type MachineControllerConfiguration struct {
	// ImageRepository is used to override the Machine Controller image repository.
	// It is only for development, tests and PoC purposes. This field must not be set in production environments.
	ImageRepository string `json:"imageRepository,omitempty"`
	// ImageTag is used to override the Machine Controller image.
	// It is only for development, tests and PoC purposes. This field must not be set in production environments.
	ImageTag string `json:"imageTag,omitempty"`
}

// OperatingSystemManager configures the image repo and the tag version for osm deployment.
type OperatingSystemManager struct {
	// ImageRepository is used to override the OperatingSystemManager image repository.
	// It is recommended to use this field only for development, tests and PoC purposes. For production environments.
	// it is not recommended, to use this field due to compatibility with the overall KKP stack.
	ImageRepository string `json:"imageRepository,omitempty"`
	// ImageTag is used to override the OperatingSystemManager image.
	// It is recommended to use this field only for development, tests and PoC purposes. For production environments.
	// it is not recommended, to use this field due to compatibility with the overall KKP stack.
	ImageTag string `json:"imageTag,omitempty"`
}

// KubermaticAddonConfiguration describes the addons for a given cluster runtime.
type KubermaticAddonsConfiguration struct {
	// Default is the list of addons to be installed by default into each cluster.
	// Mutually exclusive with "defaultManifests".
	Default []string `json:"default,omitempty"`
	// DefaultManifests is a list of addon manifests to install into all clusters.
	// Mutually exclusive with "default".
	DefaultManifests string `json:"defaultManifests,omitempty"`
	// DockerRepository is the repository containing the Docker image containing
	// the possible addon manifests.
	DockerRepository string `json:"dockerRepository,omitempty"`
	// DockerTagSuffix is appended to the tag used for referring to the addons image.
	// If left empty, the tag will be the KKP version (e.g. "v2.15.0"), with a
	// suffix it becomes "v2.15.0-SUFFIX".
	DockerTagSuffix string `json:"dockerTagSuffix,omitempty"`
}

// SystemApplicationsConfiguration contains configuration for system Applications (e.g. CNI).
type SystemApplicationsConfiguration struct {
	// HelmRepository specifies OCI repository containing Helm charts of system Applications e.g. oci://localhost:5000/myrepo.
	HelmRepository string `json:"helmRepository,omitempty"`
	// HelmRegistryConfigFile optionally holds the ref and key in the secret for the OCI registry credential file.
	// The value is dockercfg file that follows the same format rules as ~/.docker/config.json
	// The Secret must exist in the namespace where KKP is installed (default is "kubermatic").
	// The Secret must be annotated with `apps.kubermatic.k8c.io/secret-type:` set to "helm".
	HelmRegistryConfigFile *corev1.SecretKeySelector `json:"helmRegistryConfigFile,omitempty"`
}

// ApplicationsConfiguration contains configuration for default Applications configuration settings.
type ApplicationsConfiguration struct {
	// Namespace is the namespace which is set as the default for applications installed via ui
	// If left empty the default for the application installation namespace is the name of the resource itself
	Namespace string `json:"namespace,omitempty"`
}

type KubermaticIngressConfiguration struct {
	// Domain is the base domain where the dashboard shall be available. Even with
	// a disabled Ingress, this must always be a valid hostname.
	Domain string `json:"domain"`

	// ClassName is the Ingress resource's class name, used for selecting the appropriate
	// ingress controller.
	ClassName string `json:"className,omitempty"`

	// NamespaceOverride need to be set if a different ingress-controller is used than the KKP default one.
	NamespaceOverride string `json:"namespaceOverride,omitempty"`

	// Disable will prevent an Ingress from being created at all. This is mostly useful
	// during testing. If the Ingress is disabled, the CertificateIssuer setting can also
	// be left empty, as no Certificate resource will be created.
	Disable bool `json:"disable,omitempty"`

	// CertificateIssuer is the name of a cert-manager Issuer or ClusterIssuer (default)
	// that will be used to acquire the certificate for the configured domain.
	// To use a namespaced Issuer, set the Kind to "Issuer" and manually create the
	// matching Issuer in Kubermatic's namespace.
	// Setting an empty name disables the automatic creation of certificates and disables
	// the TLS settings on the Kubermatic Ingress.
	CertificateIssuer corev1.TypedLocalObjectReference `json:"certificateIssuer,omitempty"`
}

// KubermaticMasterControllerConfiguration configures the Kubermatic master controller-manager.
type KubermaticMasterControllerConfiguration struct {
	// DockerRepository is the repository containing the Kubermatic master-controller-manager image.
	DockerRepository string `json:"dockerRepository,omitempty"`
	// ProjectsMigrator configures the migrator for user projects.
	ProjectsMigrator KubermaticProjectsMigratorConfiguration `json:"projectsMigrator,omitempty"`
	// PProfEndpoint controls the port the master-controller-manager should listen on to provide pprof
	// data. This port is never exposed from the container and only available via port-forwardings.
	PProfEndpoint *string `json:"pprofEndpoint,omitempty"`
	// Resources describes the requested and maximum allowed CPU/memory usage.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// DebugLog enables more verbose logging.
	DebugLog bool `json:"debugLog,omitempty"`
	// Replicas sets the number of pod replicas for the master-controller-manager.
	Replicas *int32 `json:"replicas,omitempty"`
}

// KubermaticProjectsMigratorConfiguration configures the Kubermatic master controller-manager.
type KubermaticProjectsMigratorConfiguration struct {
	// DryRun makes the migrator only log the actions it would take.
	DryRun bool `json:"dryRun,omitempty"`
}

// KubermaticVersioningConfiguration configures the available and default Kubernetes versions.
type KubermaticVersioningConfiguration struct {
	// Versions lists the available versions.
	Versions []semver.Semver `json:"versions,omitempty"`
	// Default is the default version to offer users.
	Default *semver.Semver `json:"default,omitempty"`

	// Updates is a list of available and automatic upgrades.
	// All 'to' versions must be configured in the version list for this orchestrator.
	// Each update may optionally be configured to be 'automatic: true', in which case the
	// controlplane of all clusters whose version matches the 'from' directive will get
	// updated to the 'to' version. If automatic is enabled, the 'to' version must be a
	// version and not a version range.
	// Also, updates may set 'automaticNodeUpdate: true', in which case Nodes will get
	// updates as well. 'automaticNodeUpdate: true' implies 'automatic: true' as well,
	// because Nodes may not have a newer version than the controlplane.
	Updates []Update `json:"updates,omitempty"`

	// ProviderIncompatibilities lists all the Kubernetes version incompatibilities
	ProviderIncompatibilities []Incompatibility `json:"providerIncompatibilities,omitempty"`

	// ExternalClusters contains the available and default Kubernetes versions and updates for ExternalClusters.
	ExternalClusters map[ExternalClusterProviderType]ExternalClusterProviderVersioningConfiguration `json:"externalClusters,omitempty"`
}

// ExternalClusterProviderType is used to indicate ExternalCluster Provider Types.
type ExternalClusterProviderType string

const (
	EKSProviderType ExternalClusterProviderType = "eks"
	GKEProviderType ExternalClusterProviderType = "gke"
	AKSProviderType ExternalClusterProviderType = "aks"
)

// ExternalClusterProviderVersioningConfiguration configures the available and default Kubernetes versions for ExternalCluster Providers.
type ExternalClusterProviderVersioningConfiguration struct {
	// Versions lists the available versions.
	Versions []semver.Semver `json:"versions,omitempty"`
	// Default is the default version to offer users.
	Default *semver.Semver `json:"default,omitempty"`
	// Updates is a list of available upgrades.
	Updates []semver.Semver `json:"updates,omitempty"`
}

// Update represents an update option for a user cluster.
type Update struct {
	// From is the version from which an update is allowed. Wildcards are allowed, e.g. "1.18.*".
	From string `json:"from,omitempty"`
	// To is the version to which an update is allowed.
	// Must be a valid version if `automatic` is set to true, e.g. "1.20.13".
	// Can be a wildcard otherwise, e.g. "1.20.*".
	To string `json:"to,omitempty"`
	// Automatic controls whether this update is executed automatically
	// for the control plane of all matching user clusters.
	// ---
	//nolint:staticcheck
	//lint:ignore SA5008 omitgenyaml is used by the example-yaml-generator
	Automatic *bool `json:"automatic,omitempty,omitgenyaml"`
	// Automatic controls whether this update is executed automatically
	// for the worker nodes of all matching user clusters.
	// ---
	//nolint:staticcheck
	//lint:ignore SA5008 omitgenyaml is used by the example-yaml-generator
	AutomaticNodeUpdate *bool `json:"automaticNodeUpdate,omitempty,omitgenyaml"`
}

// Incompatibility represents a version incompatibility for a user cluster.
type Incompatibility struct {
	// Provider to which to apply the compatibility check.
	// Empty string matches all providers
	// +kubebuilder:validation:Enum="";digitalocean;hetzner;azure;vsphere;aws;openstack;packet;gcp;kubevirt;nutanix;alibaba;anexia;fake;vmwareclouddirector
	Provider string `json:"provider,omitempty"`
	// Version is the Kubernetes version that must be checked. Wildcards are allowed, e.g. "1.25.*".
	Version string `json:"version,omitempty"`
	// Condition is the cluster or datacenter condition that must be met to block a specific version
	Condition ConditionType `json:"condition,omitempty"`
	// Operation is the operation triggering the compatibility check (CREATE or UPDATE)
	Operation OperationType `json:"operation,omitempty"`
}

// KubermaticVPAConfiguration configures the Kubernetes VPA.
type KubermaticVPAConfiguration struct {
	Recommender         KubermaticVPAComponent `json:"recommender,omitempty"`
	Updater             KubermaticVPAComponent `json:"updater,omitempty"`
	AdmissionController KubermaticVPAComponent `json:"admissionController,omitempty"`
}

type KubermaticVPAComponent struct {
	// DockerRepository is the repository containing the component's image.
	DockerRepository string `json:"dockerRepository,omitempty"`
	// Resources describes the requested and maximum allowed CPU/memory usage.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// KubermaticProxyConfiguration can be used to control how the various
// Kubermatic components reach external services / the Internet. These
// settings are reflected as environment variables for the Kubermatic
// pods.
type KubermaticProxyConfiguration struct {
	// HTTP is the full URL to the proxy to use for plaintext HTTP
	// connections, e.g. "http://internalproxy.example.com:8080".
	HTTP string `json:"http,omitempty"`
	// HTTPS is the full URL to the proxy to use for encrypted HTTPS
	// connections, e.g. "http://secureinternalproxy.example.com:8080".
	HTTPS string `json:"https,omitempty"`
	// NoProxy is a comma-separated list of hostnames / network masks
	// for which no proxy shall be used. If you make use of proxies,
	// this list should contain all local and cluster-internal domains
	// and networks, e.g. "10.0.0.0/8,172.16.0.0/12,192.168.0.0/16,mydomain".
	// The operator will always prepend the following elements to this
	// list if proxying is configured (i.e. HTTP/HTTPS are not empty):
	// "127.0.0.1/8", "localhost", ".local", ".local.", "kubernetes", ".default", ".svc"
	NoProxy string `json:"noProxy,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// KubermaticConfigurationList is a collection of KubermaticConfigurations.
type KubermaticConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KubermaticConfiguration `json:"items"`
}
