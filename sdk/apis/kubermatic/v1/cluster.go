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
	"encoding/base64"
	"errors"
	"fmt"
	"slices"
	"strings"

	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// ClusterResourceName represents "Resource" defined in Kubernetes.
	ClusterResourceName = "clusters"

	// ClusterKindName represents "Kind" defined in Kubernetes.
	ClusterKindName = "Cluster"

	// CredentialPrefix is the prefix used for the secrets containing cloud provider credentials.
	CredentialPrefix = "credential"

	// ForceRestartAnnotation is key of the annotation used to restart machine deployments.
	ForceRestartAnnotation = "forceRestart"

	// PresetNameAnnotation is key of the annotation used to hold preset name if was used for the cluster creation.
	PresetNameAnnotation = "presetName"

	// PresetInvalidatedAnnotation is key of the annotation used to indicate why the preset was invalidated.
	PresetInvalidatedAnnotation = "presetInvalidated"
)

const (
	CCMMigrationNeededAnnotation = "ccm-migration.k8c.io/migration-needed"
	CSIMigrationNeededAnnotation = "csi-migration.k8c.io/migration-needed"
)

const (
	WorkerNameLabelKey         = "worker-name"
	ProjectIDLabelKey          = "project-id"
	ExternalClusterIDLabelKey  = "external-cluster-id"
	UpdatedByVPALabelKey       = "updated-by-vpa"
	IsCredentialPresetLabelKey = "is-credential-preset"

	DefaultEtcdClusterSize = 3
	MinEtcdClusterSize     = 3
	MaxEtcdClusterSize     = 9

	DefaultKonnectivityKeepaliveTime = "1m"
)

// +kubebuilder:validation:Enum=standard;basic

// Azure SKU for Load Balancers. Possible values are `basic` and `standard`.
type LBSKU string

const (
	AzureStandardLBSKU = LBSKU("standard")
	AzureBasicLBSKU    = LBSKU("basic")
)

// +kubebuilder:validation:Enum=deleted;changed
type PresetInvalidationReason string

const (
	PresetDeleted = PresetInvalidationReason("deleted")
	PresetChanged = PresetInvalidationReason("changed")
)

// ProtectedClusterLabels is a set of labels that must not be set by users on clusters,
// as they are security relevant.
var ProtectedClusterLabels = sets.New(WorkerNameLabelKey, ProjectIDLabelKey, IsCredentialPresetLabelKey)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".spec.humanReadableName",name="HumanReadableName",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.userEmail",name="Owner",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.version",name="Version",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.cloud.providerName",name="Provider",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.cloud.dc",name="Datacenter",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.pause",name="Paused",type="boolean"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// Cluster represents a Kubermatic Kubernetes Platform user cluster.
// Cluster objects exist on Seed clusters and each user cluster consists
// of a namespace containing the Kubernetes control plane and additional
// pods (like Prometheus or the machine-controller).
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the desired cluster state.
	Spec ClusterSpec `json:"spec,omitempty"`

	// Status contains reconciliation information for the cluster.
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ClusterList specifies a list of user clusters.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Cluster `json:"items"`
}

// ClusterSpec describes the desired state of a user cluster.
type ClusterSpec struct {
	// HumanReadableName is the cluster name provided by the user.
	HumanReadableName string `json:"humanReadableName"`

	// Version defines the wanted version of the control plane.
	Version semver.Semver `json:"version"`

	// Cloud contains information regarding the cloud provider that
	// is responsible for hosting the cluster's workload.
	Cloud CloudSpec `json:"cloud"`

	// +kubebuilder:validation:Enum=docker;containerd
	// +kubebuilder:default=containerd

	// ContainerRuntime to use, i.e. `docker` or `containerd`. By default `containerd` will be used.
	ContainerRuntime string `json:"containerRuntime,omitempty"`

	// Optional: ImagePullSecret references a secret with container registry credentials. This is passed to the machine-controller which sets the registry credentials on node level.
	ImagePullSecret *corev1.SecretReference `json:"imagePullSecret,omitempty"`

	// Optional: CNIPlugin refers to the spec of the CNI plugin used by the Cluster.
	CNIPlugin *CNIPluginSettings `json:"cniPlugin,omitempty"`

	// Optional: ClusterNetwork specifies the different networking parameters for a cluster.
	ClusterNetwork ClusterNetworkingConfig `json:"clusterNetwork"`

	// Optional: MachineNetworks is the list of the networking parameters used for IPAM.
	MachineNetworks []MachineNetworkingConfig `json:"machineNetworks,omitempty"`

	// ExposeStrategy is the strategy used to expose a cluster control plane.
	ExposeStrategy ExposeStrategy `json:"exposeStrategy"`

	// Optional: APIServerAllowedIPRanges is a list of IP ranges allowed to access the API server.
	// Applicable only if the expose strategy of the cluster is LoadBalancer.
	// If not configured, access to the API server is unrestricted.
	APIServerAllowedIPRanges *NetworkRanges `json:"apiServerAllowedIPRanges,omitempty"`

	// Optional: Component specific overrides that allow customization of control plane components.
	ComponentsOverride ComponentSettings `json:"componentsOverride,omitempty"`

	// Optional: OIDC specifies the OIDC configuration parameters for enabling authentication mechanism for the cluster.
	OIDC OIDCSettings `json:"oidc,omitempty"`

	// A map of optional or early-stage features that can be enabled for the user cluster.
	// Some feature gates cannot be disabled after being enabled.
	// The available feature gates vary based on KKP version, Kubernetes version and Seed configuration.
	// Please consult the KKP documentation for specific feature gates.
	Features map[string]bool `json:"features,omitempty"`

	// Optional: UpdateWindow configures automatic update systems to respect a maintenance window for
	// applying OS updates to nodes. This is only respected on Flatcar nodes currently.
	UpdateWindow *UpdateWindow `json:"updateWindow,omitempty"`

	// Enables the admission plugin `PodSecurityPolicy`. This plugin is deprecated by Kubernetes.
	UsePodSecurityPolicyAdmissionPlugin bool `json:"usePodSecurityPolicyAdmissionPlugin,omitempty"`
	// Enables the admission plugin `PodNodeSelector`. Needs additional configuration via the `podNodeSelectorAdmissionPluginConfig` field.
	UsePodNodeSelectorAdmissionPlugin bool `json:"usePodNodeSelectorAdmissionPlugin,omitempty"`
	// Enables the admission plugin `EventRateLimit`. Needs additional configuration via the `eventRateLimitConfig` field.
	// This plugin is considered "alpha" by Kubernetes.
	UseEventRateLimitAdmissionPlugin bool `json:"useEventRateLimitAdmissionPlugin,omitempty"`

	// A list of arbitrary admission plugin names that are passed to kube-apiserver. Must not include admission plugins
	// that can be enabled via a separate setting.
	AdmissionPlugins []string `json:"admissionPlugins,omitempty"`

	// Optional: Provides configuration for the PodNodeSelector admission plugin (needs plugin enabled
	// via `usePodNodeSelectorAdmissionPlugin`). It's used by the backend to create a configuration file for this plugin.
	// The key:value from this map is converted to <namespace>:<node-selectors-labels> in the file. Use `clusterDefaultNodeSelector`
	// as key to configure a default node selector.
	PodNodeSelectorAdmissionPluginConfig map[string]string `json:"podNodeSelectorAdmissionPluginConfig,omitempty"`

	// Optional: Configures the EventRateLimit admission plugin (if enabled via `useEventRateLimitAdmissionPlugin`)
	// to create limits on Kubernetes event generation. The EventRateLimit plugin is capable of comparing and rate limiting incoming
	// `Events` based on several configured buckets.
	EventRateLimitConfig *EventRateLimitConfig `json:"eventRateLimitConfig,omitempty"`

	// Optional: Deploys the UserSSHKeyAgent to the user cluster. This field is immutable.
	// If enabled, the agent will be deployed and used to sync user ssh keys attached by users to the cluster.
	// No SSH keys will be synced after node creation if this is disabled.
	EnableUserSSHKeyAgent *bool `json:"enableUserSSHKeyAgent,omitempty"`

	// Deprecated: EnableOperatingSystemManager has been deprecated starting with KKP 2.26 and will be removed in KKP 2.28+. This field is no-op and OSM is always enabled for user clusters.
	// OSM is responsible for creating and managing worker node configuration.
	EnableOperatingSystemManager *bool `json:"enableOperatingSystemManager,omitempty"`

	// KubeLB holds the configuration for the kubeLB component.
	// Only available in Enterprise Edition.
	KubeLB *KubeLB `json:"kubelb,omitempty"`

	// KubernetesDashboard holds the configuration for the kubernetes-dashboard component.
	KubernetesDashboard *KubernetesDashboard `json:"kubernetesDashboard,omitempty"`

	// Optional: AuditLogging configures Kubernetes API audit logging (https://kubernetes.io/docs/tasks/debug-application-cluster/audit/)
	// for the user cluster.
	AuditLogging *AuditLoggingSettings `json:"auditLogging,omitempty"`

	// Optional: OPAIntegration is a preview feature that enables OPA integration for the cluster.
	// Enabling it causes OPA Gatekeeper and its resources to be deployed on the user cluster.
	// By default it is disabled.
	OPAIntegration *OPAIntegrationSettings `json:"opaIntegration,omitempty"`

	// Optional: ServiceAccount contains service account related settings for the user cluster's kube-apiserver.
	ServiceAccount *ServiceAccountSettings `json:"serviceAccount,omitempty"`

	// Optional: MLA contains monitoring, logging and alerting related settings for the user cluster.
	MLA *MLASettings `json:"mla,omitempty"`

	// Optional: ApplicationSettings contains the settings relative to the application feature.
	ApplicationSettings *ApplicationSettings `json:"applicationSettings,omitempty"`

	// Optional: Configures encryption-at-rest for Kubernetes API data. This needs the `encryptionAtRest` feature gate.
	EncryptionConfiguration *EncryptionConfiguration `json:"encryptionConfiguration,omitempty"`

	// If this is set to true, the cluster will not be reconciled by KKP.
	// This indicates that the user needs to do some action to resolve the pause.
	// +kubebuilder:default=false
	Pause bool `json:"pause,omitempty"`
	// PauseReason is the reason why the cluster is not being managed. This field is for informational
	// purpose only and can be set by a user or a controller to communicate the reason for pausing the cluster.
	PauseReason string `json:"pauseReason,omitempty"`

	// Enables more verbose logging in KKP's user-cluster-controller-manager.
	DebugLog bool `json:"debugLog,omitempty"`

	// Optional: DisableCSIDriver disables the installation of CSI driver on the cluster
	// If this is true at the data center then it can't be over-written in the cluster configuration
	DisableCSIDriver bool `json:"disableCsiDriver,omitempty"`

	// Optional: BackupConfig contains the configuration options for managing the Cluster Backup Velero integration feature.
	BackupConfig *BackupConfig `json:"backupConfig,omitempty"`

	// Kyverno holds the configuration for the Kyverno policy management component.
	// Only available in Enterprise Edition.
	Kyverno *KyvernoSettings `json:"kyverno,omitempty"`

	// Optional: AuthorizationConfig to configure the apiserver authorization modes. This feature is in technical preview right now
	AuthorizationConfig *AuthorizationConfig `json:"authorizationConfig,omitempty"`

	// ContainerRuntimeOpts defines optional configuration options to configure container-runtime
	// that is going to be used in the user cluster.
	// This will not configure node level settings for container runtime used in user clusters; its only being used
	// to configure container runtime settings of a particular user cluster.
	ContainerRuntimeOpts *ContainerRuntimeOpts `json:"containerRuntimeOpts,omitempty"`
}

// KubernetesDashboard contains settings for the kubernetes-dashboard component as part of the cluster control plane.
type KubernetesDashboard struct {
	// Controls whether kubernetes-dashboard is deployed to the user cluster or not.
	// Enabled by default.
	Enabled bool `json:"enabled,omitempty"`
}

func (c ClusterSpec) IsKubernetesDashboardEnabled() bool {
	return c.KubernetesDashboard == nil || c.KubernetesDashboard.Enabled
}

// KubeLB contains settings for the kubeLB component as part of the cluster control plane. This component is responsible for managing load balancers.
// Only available in Enterprise Edition.
type KubeLB struct {
	// Controls whether kubeLB is deployed or not.
	Enabled bool `json:"enabled"`
	// UseLoadBalancerClass is used to configure the use of load balancer class `kubelb` for kubeLB. If false, kubeLB will manage all load balancers in the
	// user cluster irrespective of the load balancer class.
	UseLoadBalancerClass *bool `json:"useLoadBalancerClass,omitempty"`
	// EnableGatewayAPI is used to enable Gateway API for KubeLB. Once enabled, KubeLB installs the Gateway API CRDs in the user cluster.
	EnableGatewayAPI *bool `json:"enableGatewayAPI,omitempty"`
	// ExtraArgs are additional arbitrary flags to pass to the kubeLB CCM for the user cluster.
	ExtraArgs map[string]string `json:"extraArgs,omitempty"`
}

func (c ClusterSpec) IsKubeLBEnabled() bool {
	return c.KubeLB != nil && c.KubeLB.Enabled
}

func (k KubeLB) IsGatewayAPIEnabled() bool {
	return k.EnableGatewayAPI != nil && *k.EnableGatewayAPI
}

// CNIPluginSettings contains the spec of the CNI plugin used by the Cluster.
type CNIPluginSettings struct {
	// Type is the CNI plugin type to be used.
	Type CNIPluginType `json:"type"`
	// Version defines the CNI plugin version to be used. This varies by chosen CNI plugin type.
	Version string `json:"version"`
}

// +kubebuilder:validation:Enum=canal;cilium;none

// CNIPluginType defines the type of CNI plugin installed.
// Possible values are `canal`, `cilium` or `none`.
type CNIPluginType string

func (c CNIPluginType) String() string {
	return string(c)
}

const (
	// CNIPluginTypeCanal corresponds to Canal CNI plugin (i.e. Flannel +
	// Calico for policy enforcement).
	CNIPluginTypeCanal CNIPluginType = "canal"

	// CNIPluginTypeCilium corresponds to Cilium CNI plugin.
	CNIPluginTypeCilium CNIPluginType = "cilium"

	// CNIPluginTypeNone corresponds to no CNI plugin managed by KKP
	// (cluster users are responsible for managing the CNI in the cluster themselves).
	CNIPluginTypeNone CNIPluginType = "none"
)

const (
	// ClusterFeatureExternalCloudProvider describes the external cloud provider feature. It is
	// only supported on a limited set of providers for a specific set of Kube versions. It must
	// not be set if its not supported.
	ClusterFeatureExternalCloudProvider = "externalCloudProvider"

	// ClusterFeatureCCMClusterName sets the cluster-name flag on the external CCM deployment.
	// The cluster-name flag is often used for naming cloud resources, such as load balancers.
	ClusterFeatureCCMClusterName = "ccmClusterName"

	// ClusterFeatureVsphereCSIClusterID sets the cluster-id in the vSphere CSI config to
	// the name of the user cluster. Originally, we have been setting cluster-id to the
	// vSphere Compute Cluster name (provided via the Datacenter object), however,
	// this is supposed to identify the Kubernetes cluster, therefore it must be unique.
	// This feature flag is enabled by default for new vSphere clusters, while existing
	// vSphere clusters must be migrated manually (preferably by following advice here:
	// https://kb.vmware.com/s/article/84446).
	ClusterFeatureVsphereCSIClusterID = "vsphereCSIClusterID"

	// ClusterFeatureEtcdLauncher enables features related to the experimental etcd-launcher. This includes user-cluster
	// etcd scaling, automatic volume recovery and new backup/restore controllers.
	ClusterFeatureEtcdLauncher = "etcdLauncher"

	// ApiserverNetworkPolicy enables the deployment of network policies that
	// restrict the egress traffic from Apiserver pods.
	ApiserverNetworkPolicy = "apiserverNetworkPolicy"

	// KubeSystemNetworkPolicies enables the deployment of network policies to kube-system namespace that
	// restrict traffic from all pods in the namespace.
	KubeSystemNetworkPolicies = "kubeSystemNetworkPolicies"

	// ClusterFeatureEncryptionAtRest enables the experimental "encryption-at-rest" feature, which allows encrypting
	// Kubernetes data in etcd with a user-provided encryption key or KMS service.
	ClusterFeatureEncryptionAtRest = "encryptionAtRest"
)

// +kubebuilder:validation:Enum="";SeedResourcesUpToDate;ClusterControllerReconciledSuccessfully;AddonControllerReconciledSuccessfully;AddonInstallerControllerReconciledSuccessfully;BackupControllerReconciledSuccessfully;CloudControllerReconciledSuccessfully;UpdateControllerReconciledSuccessfully;MonitoringControllerReconciledSuccessfully;MachineDeploymentReconciledSuccessfully;MLAControllerReconciledSuccessfully;ClusterInitialized;EtcdClusterInitialized;CSIKubeletMigrationCompleted;ClusterUpdateSuccessful;ClusterUpdateInProgress;CSIKubeletMigrationSuccess;CSIKubeletMigrationInProgress;EncryptionControllerReconciledSuccessfully;IPAMControllerReconciledSuccessfully;

// ClusterConditionType is used to indicate the type of a cluster condition. For all condition
// types, the `true` value must indicate success. All condition types must be registered within
// the `AllClusterConditionTypes` variable.
type ClusterConditionType string

// UpdateWindow allows defining windows for maintenance tasks related to OS updates.
// This is only applied to cluster nodes using Flatcar Linux.
// The reference time for this is the node system time and might differ from
// the user's timezone, which needs to be considered when configuring a window.
type UpdateWindow struct {
	// Sets the start time of the update window. This can be a time of day in 24h format, e.g. `22:30`,
	// or a day of week plus a time of day, for example `Mon 21:00`. Only short names for week days are supported,
	// i.e. `Mon`, `Tue`, `Wed`, `Thu`, `Fri`, `Sat` and `Sun`.
	Start string `json:"start,omitempty"`
	// Sets the length of the update window beginning with the start time. This needs to be a valid duration
	// as parsed by Go's time.ParseDuration (https://pkg.go.dev/time#ParseDuration), e.g. `2h`.
	Length string `json:"length,omitempty"`
}

// EncryptionConfiguration configures encryption-at-rest for Kubernetes API data.
type EncryptionConfiguration struct {
	// Enables encryption-at-rest on this cluster.
	Enabled bool `json:"enabled"`

	// +kubebuilder:validation:MinItems=1

	// List of resources that will be stored encrypted in etcd.
	Resources []string `json:"resources"`
	// Configuration for the `secretbox` static key encryption scheme as supported by Kubernetes.
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/encrypt-data/#providers
	Secretbox *SecretboxEncryptionConfiguration `json:"secretbox,omitempty"`
}

// SecretboxEncryptionConfiguration defines static key encryption based on the 'secretbox' solution for Kubernetes.
type SecretboxEncryptionConfiguration struct {
	// +kubebuilder:validation:MinItems=1

	// List of 'secretbox' encryption keys. The first element of this list is considered
	// the "primary" key which will be used for encrypting data while writing it. Additional
	// keys will be used for decrypting data while reading it, if keys higher in the list
	// did not succeed in decrypting it.
	Keys []SecretboxKey `json:"keys"`
}

// SecretboxKey stores a key or key reference for encrypting Kubernetes API data at rest with a static key.
type SecretboxKey struct {
	// Identifier of a key, used in various places to refer to the key.
	Name string `json:"name"`
	// Value contains a 32-byte random key that is base64 encoded. This is the key used
	// for encryption. Can be generated via `head -c 32 /dev/urandom | base64`, for example.
	Value string `json:"value,omitempty"`
	// Instead of passing the sensitive encryption key via the `value` field, a secret can be
	// referenced. The key of the secret referenced here needs to hold a key equivalent to the `value` field.
	SecretRef *corev1.SecretKeySelector `json:"secretRef,omitempty"`
}

type BackupConfig struct {
	BackupStorageLocation *corev1.LocalObjectReference `json:"backupStorageLocation,omitempty"`
}

func (c ClusterSpec) IsClusterBackupEnabled() bool {
	return c.BackupConfig != nil &&
		c.BackupConfig.BackupStorageLocation != nil &&
		c.BackupConfig.BackupStorageLocation.Name != ""
}

// KyvernoSettings contains settings for the Kyverno component as part of the cluster control plane. This component is responsible for policy management.
type KyvernoSettings struct {
	// Controls whether Kyverno is deployed or not.
	Enabled bool `json:"enabled"`
}

func (c ClusterSpec) IsKyvernoEnabled() bool {
	return c.Kyverno != nil && c.Kyverno.Enabled
}

type AuthorizationConfig struct {
	// Optional: List of enabled Authorization modes (by default 'Node,RBAC')
	// Important: order matters
	EnabledModes []string `json:"enabledModes,omitempty"`
	// Contains the settings for the AuthorizationWebhook if EnabledModes contains Webhook
	AuthorizationWebhookConfiguration *AuthorizationWebhookConfiguration `json:"authorizationWebhookConfiguration,omitempty"`
	// Configuration options for mounting the authorization config file from a secret
	AuthorizationConfigurationFile *AuthorizationConfigurationFile `json:"authorizationConfigurationFile,omitempty"`
}

type AuthorizationWebhookConfiguration struct {
	// The secret containing the webhook configuration
	SecretName string `json:"secretName"`
	// The secret Key inside the secret
	SecretKey string `json:"secretKey"`
	// the Webhook Version, by default "v1"
	WebhookVersion string `json:"webhookVersion"`
	// Optional: The duration to cache authorization decisions for successful authorization webhook calls.
	CacheAuthorizedTTL *metav1.Duration `json:"cacheAuthorizedTTL,omitempty"`
	// Optional: The duration to cache authorization decisions for failed authorization webhook calls.
	CacheUnauthorizedTTL *metav1.Duration `json:"cacheUnauthorizedTTL,omitempty"`
}

type AuthorizationConfigurationFile struct {
	// The secret containing the authorizaion configuration
	SecretName string `json:"secretName"`
	// The secret Key containing the AuthorizationConfig k8s object
	SecretKey string `json:"secretKey"`
	// the path were the secret should be mounted, by default '/etc/kubernetes/authorization-configs'
	SecretMountPath string `json:"secretMountPath,omitempty"`
}

func (c ClusterSpec) IsWebhookAuthorizationEnabled() bool {
	if c.AuthorizationConfig == nil || c.AuthorizationConfig.EnabledModes == nil || c.AuthorizationConfig.AuthorizationWebhookConfiguration == nil {
		return false
	}

	if !slices.Contains(c.AuthorizationConfig.EnabledModes, "Webhook") {
		return false
	}

	if len(c.AuthorizationConfig.AuthorizationWebhookConfiguration.SecretName) == 0 || len(c.AuthorizationConfig.AuthorizationWebhookConfiguration.SecretKey) == 0 {
		return false
	}

	return true
}

func (c ClusterSpec) GetAuthorizationWebhookVersion() string {
	if c.AuthorizationConfig != nil && c.AuthorizationConfig.AuthorizationWebhookConfiguration != nil && len(c.AuthorizationConfig.AuthorizationWebhookConfiguration.WebhookVersion) > 0 {
		return c.AuthorizationConfig.AuthorizationWebhookConfiguration.WebhookVersion
	}

	return "v1"
}

func (c ClusterSpec) IsAuthorizationConfigurationFileEnabled() bool {
	if c.AuthorizationConfig == nil || c.AuthorizationConfig.AuthorizationConfigurationFile == nil {
		return false
	}

	if len(c.AuthorizationConfig.AuthorizationConfigurationFile.SecretName) == 0 || len(c.AuthorizationConfig.AuthorizationConfigurationFile.SecretKey) == 0 {
		return false
	}

	return true
}

const (
	// ClusterConditionSeedResourcesUpToDate indicates that all controllers have finished setting up the
	// resources for a user clusters that run inside the seed cluster, i.e. this ignores
	// the status of cloud provider resources for a given cluster.
	ClusterConditionSeedResourcesUpToDate ClusterConditionType = "SeedResourcesUpToDate"

	ClusterConditionClusterControllerReconcilingSuccess                          ClusterConditionType = "ClusterControllerReconciledSuccessfully"
	ClusterConditionAddonControllerReconcilingSuccess                            ClusterConditionType = "AddonControllerReconciledSuccessfully"
	ClusterConditionAddonInstallerControllerReconcilingSuccess                   ClusterConditionType = "AddonInstallerControllerReconciledSuccessfully"
	ClusterConditionCloudControllerReconcilingSuccess                            ClusterConditionType = "CloudControllerReconciledSuccessfully"
	ClusterConditionUpdateControllerReconcilingSuccess                           ClusterConditionType = "UpdateControllerReconciledSuccessfully"
	ClusterConditionMonitoringControllerReconcilingSuccess                       ClusterConditionType = "MonitoringControllerReconciledSuccessfully"
	ClusterConditionMachineDeploymentControllerReconcilingSuccess                ClusterConditionType = "MachineDeploymentReconciledSuccessfully"
	ClusterConditionApplicationInstallationControllerReconcilingSuccess          ClusterConditionType = "ApplicationInstallationControllerReconciledSuccessfully"
	ClusterConditionDefaultApplicationInstallationControllerReconcilingSuccess   ClusterConditionType = "DefaultApplicationInstallationControllerReconciledSuccessfully"
	ClusterConditionDefaultApplicationInstallationsControllerCreatedSuccessfully ClusterConditionType = "DefaultApplicationsCreatedSuccessfully"
	ClusterConditionOperatingSystemManagerMigratorControllerReconcilingSuccess   ClusterConditionType = "OperatingSystemManagerMigratorControllerReconciledSuccessfully"
	ClusterConditionKubeLBControllerReconcilingSuccess                           ClusterConditionType = "KubeLBControllerReconciledSuccessfully"
	ClusterConditionCNIControllerReconcilingSuccess                              ClusterConditionType = "CNIControllerReconciledSuccessfully"
	ClusterConditionMLAControllerReconcilingSuccess                              ClusterConditionType = "MLAControllerReconciledSuccessfully"
	ClusterConditionEncryptionControllerReconcilingSuccess                       ClusterConditionType = "EncryptionControllerReconciledSuccessfully"
	ClusterConditionClusterInitialized                                           ClusterConditionType = "ClusterInitialized"
	ClusterConditionIPAMControllerReconcilingSuccess                             ClusterConditionType = "IPAMControllerReconciledSuccessfully"
	ClusterConditionKubeVirtNetworkControllerSuccess                             ClusterConditionType = "KubeVirtNetworkControllerReconciledSuccessfully"
	ClusterConditionClusterBackupControllerReconcilingSuccess                    ClusterConditionType = "ClusterBackupControllerReconciledSuccessfully"
	ClusterConditionKyvernoControllerReconcilingSuccess                          ClusterConditionType = "KyvernoControllerReconciledSuccessfully"
	ClusterConditionDefaultPolicyControllerReconcilingSuccess                    ClusterConditionType = "DefaultPolicyControllerReconciledSuccessfully"
	ClusterConditionDefaultPolicyBindingsControllerCreatedSuccessfully           ClusterConditionType = "DefaultPolicyBindingsControllerCreatedSuccessfully"

	ClusterConditionEtcdClusterInitialized ClusterConditionType = "EtcdClusterInitialized"
	ClusterConditionEncryptionInitialized  ClusterConditionType = "EncryptionInitialized"

	ClusterConditionUpdateProgress ClusterConditionType = "UpdateProgress"

	// ClusterConditionNone is a special value indicating that no cluster condition should be set.
	ClusterConditionNone ClusterConditionType = ""
	// This condition is met when a CSI migration is ongoing and the CSI
	// migration feature gates are activated on the Kubelets of all the nodes.
	// When this condition is `true` CSIMigration{provider}Complete can be
	// enabled.
	ClusterConditionCSIKubeletMigrationCompleted ClusterConditionType = "CSIKubeletMigrationCompleted"

	// This condition is used to determine if the CSI addon created by KKP is in use or not.
	// This helps in ascertaining if the CSI addon can be removed from the cluster or not.
	ClusterConditionCSIAddonInUse ClusterConditionType = "CSIAddonInUse"

	ReasonClusterUpdateSuccessful             = "ClusterUpdateSuccessful"
	ReasonClusterUpdateInProgress             = "ClusterUpdateInProgress"
	ReasonClusterCSIKubeletMigrationCompleted = "CSIKubeletMigrationSuccess"
	ReasonClusterCCMMigrationInProgress       = "CSIKubeletMigrationInProgress"
)

var AllClusterConditionTypes = []ClusterConditionType{
	ClusterConditionSeedResourcesUpToDate,
	ClusterConditionClusterControllerReconcilingSuccess,
	ClusterConditionAddonControllerReconcilingSuccess,
	ClusterConditionCloudControllerReconcilingSuccess,
	ClusterConditionUpdateControllerReconcilingSuccess,
	ClusterConditionMonitoringControllerReconcilingSuccess,
}

type ClusterCondition struct {
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// KubermaticVersion current kubermatic version.
	KubermaticVersion string `json:"kubermaticVersion"`
	// Last time we got an update on a given condition.
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:validation:Enum=Creating;Updating;Running;Terminating

type ClusterPhase string

// These are the valid phases of a project.
const (
	ClusterCreating    ClusterPhase = "Creating"
	ClusterUpdating    ClusterPhase = "Updating"
	ClusterRunning     ClusterPhase = "Running"
	ClusterTerminating ClusterPhase = "Terminating"
)

// ClusterStatus stores status information about a cluster.
type ClusterStatus struct {
	// Address contains the IPs/URLs to access the cluster control plane.
	// +optional
	Address ClusterAddress `json:"address,omitempty"`

	// Deprecated: LastUpdated contains the timestamp at which the cluster was last modified.
	// It is kept only for KKP 2.20 release to not break the backwards-compatibility and not being set for KKP higher releases.
	// +optional
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
	// ExtendedHealth exposes information about the current health state.
	// Extends standard health status for new states.
	// +optional
	ExtendedHealth ExtendedClusterHealth `json:"extendedHealth,omitempty"`
	// LastProviderReconciliation is the time when the cloud provider resources
	// were last fully reconciled (during normal cluster reconciliation, KKP does
	// not re-check things like security groups, networks etc.).
	// +optional
	LastProviderReconciliation metav1.Time `json:"lastProviderReconciliation,omitempty"`
	// NamespaceName defines the namespace the control plane of this cluster is deployed in.
	// +optional
	NamespaceName string `json:"namespaceName"`

	// Versions contains information regarding the current and desired versions
	// of the cluster control plane and worker nodes.
	// +optional
	Versions ClusterVersionsStatus `json:"versions,omitempty"`

	// Deprecated: UserName contains the name of the owner of this cluster.
	// This field is not actively used and will be removed in the future.
	// +optional
	UserName string `json:"userName,omitempty"`
	// UserEmail contains the email of the owner of this cluster.
	// During cluster creation only, this field will be used to bind the `cluster-admin` `ClusterRole` to a cluster owner.
	// +optional
	UserEmail string `json:"userEmail"`

	// ErrorReason contains a error reason in case the controller encountered an error. Will be reset if the error was resolved.
	// +optional
	ErrorReason *ClusterStatusError `json:"errorReason,omitempty"`
	// ErrorMessage contains a default error message in case the controller encountered an error. Will be reset if the error was resolved.
	// +optional
	ErrorMessage *string `json:"errorMessage,omitempty"`

	// Conditions contains conditions the cluster is in, its primary use case is status signaling between controllers or between
	// controllers and the API.
	// +optional
	Conditions map[ClusterConditionType]ClusterCondition `json:"conditions,omitempty"`
	// Phase is a description of the current cluster status, summarizing the various conditions,
	// possible active updates etc. This field is for informational purpose only and no logic
	// should be tied to the phase.
	// +optional
	Phase ClusterPhase `json:"phase,omitempty"`

	// InheritedLabels are labels the cluster inherited from the project. They are read-only for users.
	// +optional
	InheritedLabels map[string]string `json:"inheritedLabels,omitempty"`

	// Encryption describes the status of the encryption-at-rest feature for encrypted data in etcd.
	// +optional
	Encryption *ClusterEncryptionStatus `json:"encryption,omitempty"`

	// ResourceUsage shows the current usage of resources for the cluster.
	ResourceUsage *ResourceDetails `json:"resourceUsage,omitempty"`
}

// ClusterVersionsStatus contains information regarding the current and desired versions
// of the cluster control plane and worker nodes.
type ClusterVersionsStatus struct {
	// ControlPlane is the currently active cluster version. This can lag behind the apiserver
	// version if an update is currently rolling out.
	ControlPlane semver.Semver `json:"controlPlane"`
	// Apiserver is the currently desired version of the kube-apiserver. During
	// upgrades across multiple minor versions (e.g. from 1.20 to 1.23), this will gradually
	// be increased by the update-controller until the desired cluster version (spec.version)
	// is reached.
	Apiserver semver.Semver `json:"apiserver"`
	// ControllerManager is the currently desired version of the kube-controller-manager. This
	// field behaves the same as the apiserver field.
	ControllerManager semver.Semver `json:"controllerManager"`
	// Scheduler is the currently desired version of the kube-scheduler. This field behaves the
	// same as the apiserver field.
	Scheduler semver.Semver `json:"scheduler"`
	// OldestNodeVersion is the oldest node version currently in use inside the cluster. This can be
	// nil if there are no nodes. This field is primarily for speeding up reconciling, so that
	// the controller doesn't have to re-fetch to the usercluster and query its node on every
	// reconciliation.
	OldestNodeVersion *semver.Semver `json:"oldestNodeVersion,omitempty"`
}

// HasConditionValue returns true if the cluster status has the given condition with the given status.
// It does not verify that the condition has been set by a certain Kubermatic version, it just checks
// the existence.
func (cs *ClusterStatus) HasConditionValue(conditionType ClusterConditionType, conditionStatus corev1.ConditionStatus) bool {
	condition, exists := cs.Conditions[conditionType]
	if !exists {
		return false
	}

	return condition.Status == conditionStatus
}

// +kubebuilder:validation:Enum=InvalidConfiguration;UnsupportedChange;ReconcileError

type ClusterStatusError string

const (
	InvalidConfigurationClusterError ClusterStatusError = "InvalidConfiguration"
	UnsupportedChangeClusterError    ClusterStatusError = "UnsupportedChange"
	ReconcileClusterError            ClusterStatusError = "ReconcileError"
)

// ClusterEncryptionStatus holds status information about the encryption-at-rest feature on the user cluster.
type ClusterEncryptionStatus struct {
	// The current "primary" key used to encrypt data written to etcd. Secondary keys that can be used for decryption
	// (but not encryption) might be configured in the ClusterSpec.
	ActiveKey string `json:"activeKey"`

	// List of resources currently encrypted.
	EncryptedResources []string `json:"encryptedResources"`

	// The current phase of the encryption process. Can be one of `Pending`, `Failed`, `Active` or `EncryptionNeeded`.
	// The `encryption_controller` logic will process the cluster based on the current phase and issue necessary changes
	// to make sure encryption on the cluster is active and updated with what the ClusterSpec defines.
	Phase ClusterEncryptionPhase `json:"phase"`
}

// +kubebuilder:validation:Enum=Pending;Failed;Active;EncryptionNeeded
type ClusterEncryptionPhase string

const (
	ClusterEncryptionPhasePending          ClusterEncryptionPhase = "Pending"
	ClusterEncryptionPhaseFailed           ClusterEncryptionPhase = "Failed"
	ClusterEncryptionPhaseActive           ClusterEncryptionPhase = "Active"
	ClusterEncryptionPhaseEncryptionNeeded ClusterEncryptionPhase = "EncryptionNeeded"
)

// OIDCSettings contains OIDC configuration parameters for enabling authentication mechanism for the cluster.
type OIDCSettings struct {
	IssuerURL      string `json:"issuerURL,omitempty"`
	ClientID       string `json:"clientID,omitempty"`
	ClientSecret   string `json:"clientSecret,omitempty"`
	UsernameClaim  string `json:"usernameClaim,omitempty"`
	GroupsClaim    string `json:"groupsClaim,omitempty"`
	RequiredClaim  string `json:"requiredClaim,omitempty"`
	ExtraScopes    string `json:"extraScopes,omitempty"`
	UsernamePrefix string `json:"usernamePrefix,omitempty"`
	GroupsPrefix   string `json:"groupsPrefix,omitempty"`
}

// EventRateLimitType defines the type of event rate limit.
type EventRateLimitType string

const (
	// EventRateLimitTypeServer is a limit where one bucket is shared by all event queries.
	EventRateLimitTypeServer EventRateLimitType = "Server"
	// EventRateLimitTypeNamespace is a limit where one bucket is used by each namespace.
	EventRateLimitTypeNamespace EventRateLimitType = "Namespace"
	// EventRateLimitTypeUser is a limit where one bucket is used by each user.
	EventRateLimitTypeUser EventRateLimitType = "User"
	// EventRateLimitTypeSourceAndObject is a limit where one bucket is used by each source+object combination.
	EventRateLimitTypeSourceAndObject EventRateLimitType = "SourceAndObject"
)

// EventRateLimitConfig configures the `EventRateLimit` admission plugin.
// More info: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/#eventratelimit
type EventRateLimitConfig struct {
	Server          *EventRateLimitConfigItem `json:"server,omitempty"`
	Namespace       *EventRateLimitConfigItem `json:"namespace,omitempty"`
	User            *EventRateLimitConfigItem `json:"user,omitempty"`
	SourceAndObject *EventRateLimitConfigItem `json:"sourceAndObject,omitempty"`
}

type EventRateLimitConfigItem struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=50
	//
	// QPS is the queries per second allowed for this limit type.
	QPS int32 `json:"qps"`

	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default=100
	//
	// Burst is the maximum burst size for this limit type.
	Burst int32 `json:"burst"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:default=4096
	// +optional
	//
	// CacheSize is the size of the LRU cache for this limit type.
	CacheSize int32 `json:"cacheSize,omitempty"`
}

// OPAIntegrationSettings configures the usage of OPA (Open Policy Agent) Gatekeeper inside the user cluster.
type OPAIntegrationSettings struct {
	// Enables OPA Gatekeeper integration.
	Enabled bool `json:"enabled,omitempty"`

	// +kubebuilder:default=10

	// The timeout in seconds that is set for the Gatekeeper validating webhook admission review calls.
	// Defaults to `10` (seconds).
	WebhookTimeoutSeconds *int32 `json:"webhookTimeoutSeconds,omitempty"`
	// Optional: Enables experimental mutation in Gatekeeper.
	ExperimentalEnableMutation bool `json:"experimentalEnableMutation,omitempty"`
	// Optional: ControllerResources is the resource requirements for user cluster gatekeeper controller.
	ControllerResources *corev1.ResourceRequirements `json:"controllerResources,omitempty"`
	// Optional: AuditResources is the resource requirements for user cluster gatekeeper audit.
	AuditResources *corev1.ResourceRequirements `json:"auditResources,omitempty"`
}

type ServiceAccountSettings struct {
	TokenVolumeProjectionEnabled bool `json:"tokenVolumeProjectionEnabled,omitempty"`
	// Issuer is the identifier of the service account token issuer
	// If this is not specified, it will be set to the URL of apiserver by default
	Issuer string `json:"issuer,omitempty"`
	// APIAudiences are the Identifiers of the API
	// If this is not specified, it will be set to a single element list containing the issuer URL
	APIAudiences []string `json:"apiAudiences,omitempty"`
}

type MLASettings struct {
	// MonitoringEnabled is the flag for enabling monitoring in user cluster.
	MonitoringEnabled bool `json:"monitoringEnabled,omitempty"`
	// LoggingEnabled is the flag for enabling logging in user cluster.
	LoggingEnabled bool `json:"loggingEnabled,omitempty"`
	// MonitoringResources is the resource requirements for user cluster prometheus.
	MonitoringResources *corev1.ResourceRequirements `json:"monitoringResources,omitempty"`
	// LoggingResources is the resource requirements for user cluster promtail.
	LoggingResources *corev1.ResourceRequirements `json:"loggingResources,omitempty"`
	// MonitoringReplicas is the number of desired pods of user cluster prometheus deployment.
	MonitoringReplicas *int32 `json:"monitoringReplicas,omitempty"`
}

type ApplicationSettings struct {
	// CacheSize is the size of the cache used to download application's sources.
	CacheSize *resource.Quantity `json:"cacheSize,omitempty"`
}

// +kubebuilder:validation:Enum="";preferred;required

// AntiAffinityType is the type of anti-affinity that should be used. Can be "preferred"
// or "required".
type AntiAffinityType string

const (
	AntiAffinityTypePreferred = "preferred"
	AntiAffinityTypeRequired  = "required"
)

type ComponentSettings struct {
	// Apiserver configures kube-apiserver settings.
	Apiserver APIServerSettings `json:"apiserver,omitempty"`
	// ControllerManager configures kube-controller-manager settings.
	ControllerManager ControllerSettings `json:"controllerManager,omitempty"`
	// Scheduler configures kube-scheduler settings.
	Scheduler ControllerSettings `json:"scheduler,omitempty"`
	// Etcd configures the etcd ring used to store Kubernetes data.
	Etcd EtcdStatefulSetSettings `json:"etcd,omitempty"`
	// Prometheus configures the Prometheus instance deployed into the cluster control plane.
	Prometheus StatefulSetSettings `json:"prometheus,omitempty"`
	// NodePortProxyEnvoy configures the per-cluster nodeport-proxy-envoy that is deployed if
	// the `LoadBalancer` expose strategy is used. This is not effective if a different expose
	// strategy is configured.
	NodePortProxyEnvoy NodeportProxyComponent `json:"nodePortProxyEnvoy,omitempty"`
	// KonnectivityProxy configures konnectivity-server and konnectivity-agent components.
	KonnectivityProxy KonnectivityProxySettings `json:"konnectivityProxy,omitempty"`
	// UserClusterController configures the KKP usercluster-controller deployed as part of the cluster control plane.
	UserClusterController *ControllerSettings `json:"userClusterController,omitempty"`
	// OperatingSystemManager configures operating-system-manager (the component generating node bootstrap scripts for machine-controller).
	OperatingSystemManager *OSMControllerSettings `json:"operatingSystemManager,omitempty"`
	// CoreDNS configures CoreDNS deployed as part of the cluster control plane.
	CoreDNS *DeploymentSettings `json:"coreDNS,omitempty"`
	// KubeStateMetrics configures kube-state-metrics settings deployed by the monitoring controller.
	KubeStateMetrics *DeploymentSettings `json:"kubeStateMetrics,omitempty"`
	// MachineController configures the Kubermatic machine-controller deployment.
	MachineController *DeploymentSettings `json:"machineController,omitempty"`
	// EnvoyAgent configures the envoy-agent deployed in the usercluster.
	EnvoyAgent *DaemonSetSettings `json:"envoyAgent,omitempty"`
}

type APIServerSettings struct {
	DeploymentSettings `json:",inline"`

	EndpointReconcilingDisabled *bool  `json:"endpointReconcilingDisabled,omitempty"`
	NodePortRange               string `json:"nodePortRange,omitempty"`
}

type KonnectivityProxySettings struct {
	// Resources configure limits/requests for Konnectivity components.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// KeepaliveTime represents a duration of time to check if the transport is still alive.
	// The option is propagated to agents and server.
	// Defaults to 1m.
	KeepaliveTime string `json:"keepaliveTime,omitempty"`
	// Args configures arguments (flags) for the Konnectivity deployments.
	Args []string `json:"args,omitempty"`
}

type OSMControllerSettings struct {
	ControllerSettings `json:",inline"`
	// ProxySettings defines optional flags for OperatingSystemManager deployment to allow
	// setting specific proxy configurations for specific user clusters.
	Proxy ProxySettings `json:"proxy,omitempty"`
}

type ControllerSettings struct {
	DeploymentSettings     `json:",inline"`
	LeaderElectionSettings `json:"leaderElection,omitempty"`
}

type DeploymentSettings struct {
	Replicas    *int32                       `json:"replicas,omitempty"`
	Resources   *corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration          `json:"tolerations,omitempty"`
}

type DaemonSetSettings struct {
	Resources   *corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration          `json:"tolerations,omitempty"`
}

type StatefulSetSettings struct {
	Replicas    *int32                       `json:"replicas,omitempty"`
	Resources   *corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration          `json:"tolerations,omitempty"`
}

type EtcdStatefulSetSettings struct {
	// ClusterSize is the number of replicas created for etcd. This should be an
	// odd number to guarantee consensus, e.g. 3, 5 or 7.
	ClusterSize *int32 `json:"clusterSize,omitempty"`
	// StorageClass is the Kubernetes StorageClass used for persistent storage
	// which stores the etcd WAL and other data persisted across restarts. Defaults to
	// `kubermatic-fast` (the global default).
	StorageClass string `json:"storageClass,omitempty"`
	// DiskSize is the volume size used when creating persistent storage from
	// the configured StorageClass. This is inherited from KubermaticConfiguration
	// if not set. Defaults to 5Gi.
	DiskSize *resource.Quantity `json:"diskSize,omitempty"`
	// Resources allows to override the resource requirements for etcd Pods.
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Tolerations allows to override the scheduling tolerations for etcd Pods.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// HostAntiAffinity allows to enforce a certain type of host anti-affinity on etcd
	// pods. Options are "preferred" (default) and "required". Please note that
	// enforcing anti-affinity via "required" can mean that pods are never scheduled.
	HostAntiAffinity AntiAffinityType `json:"hostAntiAffinity,omitempty"`
	// ZoneAntiAffinity allows to enforce a certain type of availability zone anti-affinity on etcd
	// pods. Options are "preferred" (default) and "required". Please note that
	// enforcing anti-affinity via "required" can mean that pods are never scheduled.
	ZoneAntiAffinity AntiAffinityType `json:"zoneAntiAffinity,omitempty"`
	// NodeSelector is a selector which restricts the set of nodes where etcd Pods can run.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// QuotaBackendGB is the maximum backend size of etcd in GB (0 means use etcd default).
	//
	// For more details, please see https://etcd.io/docs/v3.5/op-guide/maintenance/
	QuotaBackendGB *int64 `json:"quotaBackendGb,omitempty"`
}

type LeaderElectionSettings struct {
	// LeaseDurationSeconds is the duration in seconds that non-leader candidates
	// will wait to force acquire leadership. This is measured against time of
	// last observed ack.
	// +optional
	LeaseDurationSeconds *int32 `json:"leaseDurationSeconds,omitempty"`
	// RenewDeadlineSeconds is the duration in seconds that the acting controlplane
	// will retry refreshing leadership before giving up.
	// +optional
	RenewDeadlineSeconds *int32 `json:"renewDeadlineSeconds,omitempty"`
	// RetryPeriodSeconds is the duration in seconds the LeaderElector clients
	// should wait between tries of actions.
	// +optional
	RetryPeriodSeconds *int32 `json:"retryPeriodSeconds,omitempty"`
}

// +kubebuilder:validation:Enum="";IPv4;IPv4+IPv6
type IPFamily string

const (
	// IPFamilyUnspecified represents unspecified IP address family, which is interpreted as IPv4.
	IPFamilyUnspecified IPFamily = ""
	// IPFamilyIPv4 represents IPv4-only address family.
	IPFamilyIPv4 IPFamily = "IPv4"
	// IPFamilyDualStack represents dual-stack address family with IPv4 as the primary address family.
	IPFamilyDualStack IPFamily = "IPv4+IPv6"
)

// ClusterNetworkingConfig specifies the different networking
// parameters for a cluster.
type ClusterNetworkingConfig struct {
	// Optional: IP family used for cluster networking. Supported values are "", "IPv4" or "IPv4+IPv6".
	// Can be omitted / empty if pods and services network ranges are specified.
	// In that case it defaults according to the IP families of the provided network ranges.
	// If neither ipFamily nor pods & services network ranges are specified, defaults to "IPv4".
	// +optional
	IPFamily IPFamily `json:"ipFamily,omitempty"`

	// The network ranges from which service VIPs are allocated.
	// It can contain one IPv4 and/or one IPv6 CIDR.
	// If both address families are specified, the first one defines the primary address family.
	Services NetworkRanges `json:"services"`

	// The network ranges from which POD networks are allocated.
	// It can contain one IPv4 and/or one IPv6 CIDR.
	// If both address families are specified, the first one defines the primary address family.
	Pods NetworkRanges `json:"pods"`

	// NodeCIDRMaskSizeIPv4 is the mask size used to address the nodes within provided IPv4 Pods CIDR.
	// It has to be larger than the provided IPv4 Pods CIDR. Defaults to 24.
	// +optional
	NodeCIDRMaskSizeIPv4 *int32 `json:"nodeCidrMaskSizeIPv4,omitempty"`

	// NodeCIDRMaskSizeIPv6 is the mask size used to address the nodes within provided IPv6 Pods CIDR.
	// It has to be larger than the provided IPv6 Pods CIDR. Defaults to 64.
	// +optional
	NodeCIDRMaskSizeIPv6 *int32 `json:"nodeCidrMaskSizeIPv6,omitempty"`

	// Domain name for services.
	DNSDomain string `json:"dnsDomain"`

	// +kubebuilder:validation:Enum=ipvs;iptables;ebpf
	// +kubebuilder:default=ipvs

	// ProxyMode defines the kube-proxy mode ("ipvs" / "iptables" / "ebpf").
	// Defaults to "ipvs". "ebpf" disables kube-proxy and requires CNI support.
	ProxyMode string `json:"proxyMode"`

	// IPVS defines kube-proxy ipvs configuration options
	IPVS *IPVSConfiguration `json:"ipvs,omitempty"`

	// +kubebuilder:default=true

	// NodeLocalDNSCacheEnabled controls whether the NodeLocal DNS Cache feature is enabled.
	// Defaults to true.
	NodeLocalDNSCacheEnabled *bool `json:"nodeLocalDNSCacheEnabled,omitempty"`

	// CoreDNSReplicas is the number of desired pods of user cluster coredns deployment.
	// Deprecated: This field should not be used anymore, use cluster.componentsOverride.coreDNS.replicas
	// instead. Only one of the two fields can be set at any time.
	CoreDNSReplicas *int32 `json:"coreDNSReplicas,omitempty"`

	// +kubebuilder:default=true

	// Deprecated: KonnectivityEnabled enables konnectivity for controlplane to node network communication.
	// Konnectivity is the only supported choice for controlplane to node network communication. This field is
	// defaulted to true and setting it to false is rejected. It will be removed in a future release.
	KonnectivityEnabled *bool `json:"konnectivityEnabled,omitempty"`

	// TunnelingAgentIP is the address used by the tunneling agents
	TunnelingAgentIP string `json:"tunnelingAgentIP,omitempty"`
}

// MachineNetworkingConfig specifies the networking parameters used for IPAM.
type MachineNetworkingConfig struct {
	CIDR       string   `json:"cidr"`
	Gateway    string   `json:"gateway"`
	DNSServers []string `json:"dnsServers"`
}

// NetworkRanges represents ranges of network addresses.
type NetworkRanges struct {
	CIDRBlocks []string `json:"cidrBlocks"`
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	// URL under which the Apiserver is available
	// +optional
	URL string `json:"url"`
	// Port is the port the API server listens on
	// +optional
	Port int32 `json:"port"`
	// ExternalName is the DNS name for this cluster
	// +optional
	ExternalName string `json:"externalName"`
	// InternalName is the seed cluster internal absolute DNS name to the API server
	// +optional
	InternalName string `json:"internalURL"`
	// AdminToken is the token for the kubeconfig, the user can download
	// +optional
	AdminToken string `json:"adminToken"`
	// IP is the external IP under which the apiserver is available
	// +optional
	IP string `json:"ip"`
	// APIServerExternalAddress is the external address of the API server (IP or DNS name)
	// This field is populated only when the API server service is of type LoadBalancer. If set, this address will be used in the
	// kubeconfig for the user cluster that can be downloaded from the KKP UI.
	// +optional
	APIServerExternalAddress string `json:"apiServerExternalAddress,omitempty"`
}

// IPVSConfiguration contains ipvs-related configuration details for kube-proxy.
type IPVSConfiguration struct {
	// +kubebuilder:default=true

	// StrictArp configure arp_ignore and arp_announce to avoid answering ARP queries from kube-ipvs0 interface.
	// defaults to true.
	StrictArp *bool `json:"strictArp,omitempty"`
}

// CloudSpec stores configuration options for a given cloud provider. Provider specs are mutually exclusive.
type CloudSpec struct {
	// DatacenterName states the name of a cloud provider "datacenter" (defined in `Seed` resources)
	// this cluster should be deployed into.
	DatacenterName string `json:"dc"`

	// ProviderName is the name of the cloud provider used for this cluster.
	// This must match the given provider spec (e.g. if the providerName is
	// "aws", then the `aws` field must be set).
	ProviderName string `json:"providerName"`

	// Fake is a dummy cloud provider that is only used for testing purposes.
	// Do not try to actually use it.
	Fake *FakeCloudSpec `json:"fake,omitempty"`
	// Digitalocean defines the configuration data of the DigitalOcean cloud provider.
	Digitalocean *DigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	// Baremetal defines the configuration data for a Baremetal cluster.
	Baremetal *BaremetalCloudSpec `json:"baremetal,omitempty"`
	// BringYourOwn defines the configuration data for a Bring Your Own cluster.
	BringYourOwn *BringYourOwnCloudSpec `json:"bringyourown,omitempty"`
	// Edge defines the configuration data for an edge cluster.
	Edge *EdgeCloudSpec `json:"edge,omitempty"`
	// AWS defines the configuration data of the Amazon Web Services(AWS) cloud provider.
	AWS *AWSCloudSpec `json:"aws,omitempty"`
	// Azure defines the configuration data of the Microsoft Azure cloud.
	Azure *AzureCloudSpec `json:"azure,omitempty"`
	// Openstack defines the configuration data of an OpenStack cloud.
	Openstack *OpenstackCloudSpec `json:"openstack,omitempty"`
	// Hetzner defines the configuration data of the Hetzner cloud.
	Hetzner *HetznerCloudSpec `json:"hetzner,omitempty"`
	// VSphere defines the configuration data of the vSphere.
	VSphere *VSphereCloudSpec `json:"vsphere,omitempty"`
	// GCP defines the configuration data of the Google Cloud Platform(GCP).
	GCP *GCPCloudSpec `json:"gcp,omitempty"`
	// Kubevirt defines the configuration data of the KubeVirt.
	Kubevirt *KubevirtCloudSpec `json:"kubevirt,omitempty"`
	// Alibaba defines the configuration data of the Alibaba.
	Alibaba *AlibabaCloudSpec `json:"alibaba,omitempty"`
	// Anexia defines the configuration data of the Anexia.
	Anexia *AnexiaCloudSpec `json:"anexia,omitempty"`
	// Nutanix defines the configuration data of the Nutanix.
	Nutanix *NutanixCloudSpec `json:"nutanix,omitempty"`
	// VMwareCloudDirector defines the configuration data of the VMware Cloud Director.
	VMwareCloudDirector *VMwareCloudDirectorCloudSpec `json:"vmwareclouddirector,omitempty"`
}

// FakeCloudSpec specifies access data for a fake cloud.
type FakeCloudSpec struct {
	Token string `json:"token,omitempty"`
}

// DigitaloceanCloudSpec specifies access data to DigitalOcean.
type DigitaloceanCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// Token is used to authenticate with the DigitalOcean API.
	Token string `json:"token,omitempty"`
}

// HetznerCloudSpec specifies access data to hetzner cloud.
type HetznerCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// Token is used to authenticate with the Hetzner cloud API.
	Token string `json:"token,omitempty"`
	// Network is the pre-existing Hetzner network in which the machines are running.
	// While machines can be in multiple networks, a single one must be chosen for the
	// HCloud CCM to work.
	// If this is empty, the network configured on the datacenter will be used.
	Network string `json:"network,omitempty"`
}

// AzureCloudSpec defines cloud resource references for Microsoft Azure.
type AzureCloudSpec struct {
	// CredentialsReference allows referencing a `Secret` resource instead of passing secret data in this spec.
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// The Azure Active Directory Tenant used for this cluster.
	// Can be read from `credentialsReference` instead.
	TenantID string `json:"tenantID,omitempty"`
	// The Azure Subscription used for this cluster.
	// Can be read from `credentialsReference` instead.
	SubscriptionID string `json:"subscriptionID,omitempty"`
	// The service principal used to access Azure.
	// Can be read from `credentialsReference` instead.
	ClientID string `json:"clientID,omitempty"`
	// The client secret corresponding to the given service principal.
	// Can be read from `credentialsReference` instead.
	ClientSecret string `json:"clientSecret,omitempty"`

	// The resource group that will be used to look up and create resources for the cluster in.
	// If set to empty string at cluster creation, a new resource group will be created and this field will be updated to
	// the generated resource group's name.
	ResourceGroup string `json:"resourceGroup"`
	// Optional: Defines a second resource group that will be used for VNet related resources instead.
	// If left empty, NO additional resource group will be created and all VNet related resources use the resource group defined by `resourceGroup`.
	VNetResourceGroup string `json:"vnetResourceGroup"`
	// The name of the VNet resource used for setting up networking in.
	// If set to empty string at cluster creation, a new VNet will be created and this field will be updated to
	// the generated VNet's name.
	VNetName string `json:"vnet"`
	// The name of a subnet in the VNet referenced by `vnet`.
	// If set to empty string at cluster creation, a new subnet will be created and this field will be updated to
	// the generated subnet's name. If no VNet is defined at cluster creation, this field should be empty as well.
	SubnetName string `json:"subnet"`
	// The name of a route table associated with the subnet referenced by `subnet`.
	// If set to empty string at cluster creation, a new route table will be created and this field will be updated to
	// the generated route table's name. If no subnet is defined at cluster creation, this field should be empty as well.
	RouteTableName string `json:"routeTable"`
	// The name of a security group associated with the subnet referenced by `subnet`.
	// If set to empty string at cluster creation, a new security group will be created and this field will be updated to
	// the generated security group's name. If no subnet is defined at cluster creation, this field should be empty as well.
	SecurityGroup string `json:"securityGroup"`
	// A CIDR range that will be used to allow access to the node port range in the security group to. Only applies if
	// the security group is generated by KKP and not preexisting.
	// If NodePortsAllowedIPRange nor NodePortsAllowedIPRanges is set, the node port range can be accessed from anywhere.
	NodePortsAllowedIPRange string `json:"nodePortsAllowedIPRange,omitempty"`
	// Optional: CIDR ranges that will be used to allow access to the node port range in the security group to. Only applies if
	// the security group is generated by KKP and not preexisting.
	// If NodePortsAllowedIPRange nor NodePortsAllowedIPRanges is set,  the node port range can be accessed from anywhere.
	NodePortsAllowedIPRanges *NetworkRanges `json:"nodePortsAllowedIPRanges,omitempty"`
	// Optional: AssignAvailabilitySet determines whether KKP creates and assigns an AvailabilitySet to machines.
	// Defaults to `true` internally if not set.
	AssignAvailabilitySet *bool `json:"assignAvailabilitySet,omitempty"`
	// An availability set that will be associated with nodes created for this cluster. If this field is set to empty string
	// at cluster creation and `AssignAvailabilitySet` is set to `true`, a new availability set will be created and this field
	// will be updated to the generated availability set's name.
	AvailabilitySet string `json:"availabilitySet"`
	// LoadBalancerSKU sets the LB type that will be used for the Azure cluster, possible values are "basic" and "standard", if empty, "standard" will be used.
	LoadBalancerSKU LBSKU `json:"loadBalancerSKU"` //nolint:tagliatelle
}

// VSphereCredentials credentials represents a credential for accessing vSphere.
type VSphereCredentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// VSphereCloudSpec specifies access data to VSphere cloud.
type VSphereCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// The vSphere user name.
	// +optional
	Username string `json:"username"`
	// The vSphere user password.
	// +optional
	Password string `json:"password"`
	// The name of the vSphere network.
	// Deprecated: Use networks instead.
	// +optional
	VMNetName string `json:"vmNetName,omitempty"`
	// List of vSphere networks.
	// +optional
	Networks []string `json:"networks,omitempty"`
	// Folder to be used to group the provisioned virtual
	// machines.
	// +optional
	Folder string `json:"folder"`
	// Optional: BasePath configures a vCenter folder path that KKP will create an individual cluster folder in.
	// If it's an absolute path, the RootPath configured in the datacenter will be ignored. If it is a relative path,
	// the BasePath part will be appended to the RootPath to construct the full path. For both cases,
	// the full folder structure needs to exist. KKP will only try to create the cluster folder.
	// +optional
	BasePath string `json:"basePath,omitempty"`

	// If both Datastore and DatastoreCluster are not specified the virtual
	// machines are stored in the `DefaultDatastore` specified for the
	// Datacenter.

	// Datastore to be used for storing virtual machines and as a default for
	// dynamic volume provisioning, it is mutually exclusive with
	// DatastoreCluster.
	// +optional
	Datastore string `json:"datastore,omitempty"`
	// DatastoreCluster to be used for storing virtual machines, it is mutually
	// exclusive with Datastore.
	// +optional
	DatastoreCluster string `json:"datastoreCluster,omitempty"`

	// StoragePolicy to be used for storage provisioning
	StoragePolicy string `json:"storagePolicy"`

	// ResourcePool is used to manage resources such as cpu and memory for vSphere virtual machines. The resource pool
	// should be defined on vSphere cluster level.
	// +optional
	ResourcePool string `json:"resourcePool,omitempty"`

	// This user will be used for everything except cloud provider functionality
	InfraManagementUser VSphereCredentials `json:"infraManagementUser"`

	// Tags represents the tags that are attached or created on the cluster level, that are then propagated down to the
	// MachineDeployments. In order to attach tags on MachineDeployment, users must create the tag on a cluster level first
	// then attach that tag on the MachineDeployment.
	Tags *VSphereTag `json:"tags,omitempty"`
}

// VSphereTag represents the tags that are attached or created on the cluster level, that are then propagated down to the
// MachineDeployments. In order to attach tags on MachineDeployment, users must create the tag on a cluster level first
// then attach that tag on the MachineDeployment.
type VSphereTag struct {
	// Tags represents the name of the created tags.
	Tags []string `json:"tags"`
	// CategoryID is the id of the vsphere category that the tag belongs to. If the category id is left empty, the default
	// category id for the cluster will be used.
	CategoryID string `json:"categoryID,omitempty"`
}

// VMwareCloudDirectorCloudSpec specifies access data to VMware Cloud Director cloud.
type VMwareCloudDirectorCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// The VMware Cloud Director user name.
	// +optional
	Username string `json:"username,omitempty"`

	// The VMware Cloud Director user password.
	// +optional
	Password string `json:"password,omitempty"`

	// The VMware Cloud Director API token.
	// +optional
	APIToken string `json:"apiToken,omitempty"`

	// The name of organization to use.
	// +optional
	Organization string `json:"organization,omitempty"`

	// The organizational virtual data center.
	// +optional
	VDC string `json:"vdc,omitempty"`

	// The name of organizational virtual data center network that will be associated with the VMs and vApp.
	// Deprecated: OVDCNetwork has been deprecated starting with KKP 2.25 and will be removed in KKP 2.27+. It is recommended to use OVDCNetworks instead.
	OVDCNetwork string `json:"ovdcNetwork,omitempty"`

	// OVDCNetworks is the list of organizational virtual data center networks that will be attached to the vApp and can be consumed the VMs.
	OVDCNetworks []string `json:"ovdcNetworks,omitempty"`

	// VApp used for isolation of VMs and their associated network
	// +optional
	VApp string `json:"vapp,omitempty"`

	// Config for CSI driver
	CSI *VMwareCloudDirectorCSIConfig `json:"csi"`
}

type VMwareCloudDirectorCSIConfig struct {
	// The name of the storage profile to use for disks created by CSI driver
	StorageProfile string `json:"storageProfile"`

	// Filesystem to use for named disks, defaults to "ext4"
	// +optional
	Filesystem string `json:"filesystem,omitempty"`
}

// BaremetalCloudSpec specifies access data for a baremetal cluster.
type BaremetalCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`
	Tinkerbell           *TinkerbellCloudSpec                    `json:"tinkerbell,omitempty"`
}

type TinkerbellCloudSpec struct {
	// The cluster's kubeconfig file, encoded with base64.
	Kubeconfig string `json:"kubeconfig,omitempty"`
}

// BringYourOwnCloudSpec specifies access data for a bring your own cluster.
type BringYourOwnCloudSpec struct{}

// EdgeCloudSpec specifies access data for an edge cluster.
type EdgeCloudSpec struct{}

// AWSCloudSpec specifies access data to Amazon Web Services.
type AWSCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// The Access key ID used to authenticate against AWS.
	AccessKeyID string `json:"accessKeyID,omitempty"`
	// The Secret Access Key used to authenticate against AWS.
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	// Defines the ARN for an IAM role that should be assumed when handling resources on AWS. It will be used
	// to acquire temporary security credentials using an STS AssumeRole API operation whenever creating an AWS session.
	// +optional
	AssumeRoleARN string `json:"assumeRoleARN,omitempty"` //nolint:tagliatelle
	// An arbitrary string that may be needed when calling the STS AssumeRole API operation.
	// Using an external ID can help to prevent the "confused deputy problem".
	// +optional
	AssumeRoleExternalID string `json:"assumeRoleExternalID,omitempty"`
	VPCID                string `json:"vpcID"`
	// The IAM role, the control plane will use. The control plane will perform an assume-role
	ControlPlaneRoleARN string `json:"roleARN"` //nolint:tagliatelle
	RouteTableID        string `json:"routeTableID"`
	InstanceProfileName string `json:"instanceProfileName"`
	SecurityGroupID     string `json:"securityGroupID"`
	// A CIDR range that will be used to allow access to the node port range in the security group to. Only applies if
	// the security group is generated by KKP and not preexisting.
	// If NodePortsAllowedIPRange nor NodePortsAllowedIPRanges is set, the node port range can be accessed from anywhere.
	NodePortsAllowedIPRange string `json:"nodePortsAllowedIPRange,omitempty"`
	// Optional: CIDR ranges that will be used to allow access to the node port range in the security group to. Only applies if
	// the security group is generated by KKP and not preexisting.
	// If NodePortsAllowedIPRange nor NodePortsAllowedIPRanges is set,  the node port range can be accessed from anywhere.
	NodePortsAllowedIPRanges *NetworkRanges `json:"nodePortsAllowedIPRanges,omitempty"`
	// DisableIAMReconciling is used to disable reconciliation for IAM related configuration. This is useful in air-gapped
	// setups where access to IAM service is not possible.
	DisableIAMReconciling bool `json:"disableIAMReconciling,omitempty"` //nolint:tagliatelle
}

// OpenstackCloudSpec specifies access data to an OpenStack cloud.
type OpenstackCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// project, formally known as tenant.
	Project string `json:"project,omitempty"`
	// project id, formally known as tenantID.
	ProjectID string `json:"projectID,omitempty"`

	// Domain holds the name of the identity service (keystone) domain.
	Domain string `json:"domain,omitempty"`
	// Application credential ID to authenticate in combination with an application credential secret (which is not the user's password).
	ApplicationCredentialID string `json:"applicationCredentialID,omitempty"`
	// Application credential secret (which is not the user's password) to authenticate in combination with an application credential ID.
	ApplicationCredentialSecret string `json:"applicationCredentialSecret,omitempty"`
	UseToken                    bool   `json:"useToken,omitempty"`
	// Used internally during cluster creation
	Token string `json:"token,omitempty"`

	// Network holds the name of the internal network
	// When specified, all worker nodes will be attached to this network. If not specified, a network, subnet & router will be created.
	//
	// Note that the network is internal if the "External" field is set to false
	Network string `json:"network"`
	// SecurityGroups is the name of the security group (only supports a singular security group) that will be used for Machines in the cluster.
	// If this field is left empty, a default security group will be created and used.
	SecurityGroups string `json:"securityGroups"`
	// A CIDR range that will be used to allow access to the node port range in the security group to. Only applies if
	// the security group is generated by KKP and not preexisting.
	// If NodePortsAllowedIPRange nor NodePortsAllowedIPRanges is set, the node port range can be accessed from anywhere.
	NodePortsAllowedIPRange string `json:"nodePortsAllowedIPRange,omitempty"`
	// Optional: CIDR ranges that will be used to allow access to the node port range in the security group to. Only applies if
	// the security group is generated by KKP and not preexisting.
	// If NodePortsAllowedIPRange nor NodePortsAllowedIPRanges is set, the node port range can be accessed from anywhere.
	NodePortsAllowedIPRanges *NetworkRanges `json:"nodePortsAllowedIPRanges,omitempty"`
	// FloatingIPPool holds the name of the public network
	// The public network is reachable from the outside world
	// and should provide the pool of IP addresses to choose from.
	//
	// When specified, all worker nodes will receive a public ip from this floating ip pool
	//
	// Note that the network is external if the "External" field is set to true
	FloatingIPPool string `json:"floatingIPPool"`
	RouterID       string `json:"routerID"`
	SubnetID       string `json:"subnetID"`
	// SubnetCIDR is the CIDR that will be assigned to the subnet that is created for the cluster if the cluster spec
	// didn't specify a subnet id.
	// +optional
	SubnetCIDR string `json:"subnetCidr,omitempty"`
	// SubnetAllocationPool represents a pool of usable IPs that can be assigned to resources via the DHCP. The format is
	// first usable ip and last usable ip separated by a dash(e.g: 10.10.0.1-10.10.0.254)
	// +optional
	SubnetAllocationPool string `json:"subnetAllocationPool,omitempty"`
	// IPv6SubnetCIDR is the CIDR that will be assigned to the subnet that is created for the cluster if the cluster spec
	// didn't specify a subnet id for the IPv6 networking.
	// +optional
	IPv6SubnetCIDR string `json:"ipv6SubnetCidr,omitempty"`
	// IPv6SubnetID holds the ID of the subnet used for IPv6 networking.
	// If not provided, a new subnet will be created if IPv6 is enabled.
	// +optional
	IPv6SubnetID string `json:"ipv6SubnetID,omitempty"`
	// IPv6SubnetPool holds the name of the subnet pool used for creating new IPv6 subnets.
	// If not provided, the default IPv6 subnet pool will be used.
	// +optional
	IPv6SubnetPool string `json:"ipv6SubnetPool,omitempty"`
	// Whether or not to use Octavia for LoadBalancer type of Service
	// implementation instead of using Neutron-LBaaS.
	// Attention:Openstack CCM use Octavia as default load balancer
	// implementation since v1.17.0
	//
	// Takes precedence over the 'use_octavia' flag provided at datacenter
	// level if both are specified.
	// +optional
	UseOctavia *bool `json:"useOctavia,omitempty"`

	// Enable the `enable-ingress-hostname` cloud provider option on the Openstack CCM. Can only be used with the
	// external CCM and might be deprecated and removed in future versions as it is considered a workaround for the PROXY
	// protocol to preserve client IPs.
	// +optional
	EnableIngressHostname *bool `json:"enableIngressHostname,omitempty"`
	// Set a specific suffix for the hostnames used for the PROXY protocol workaround that is enabled by EnableIngressHostname.
	// The suffix is set to `nip.io` by default. Can only be used with the external CCM and might be deprecated and removed in
	// future versions as it is considered a workaround only.
	IngressHostnameSuffix *string `json:"ingressHostnameSuffix,omitempty"`

	// Flag to configure enablement of topology support for the Cinder CSI plugin.
	// This requires Nova and Cinder to have matching availability zones configured.
	// +optional
	CinderTopologyEnabled bool `json:"cinderTopologyEnabled,omitempty"`
	// List of LoadBalancerClass configurations to be used for the OpenStack cloud provider.
	// +optional
	LoadBalancerClasses []LoadBalancerClass `json:"loadBalancerClasses,omitempty"`
	// NodeVolumeAttachLimit defines the maximum number of volumes that can be
	// attached to a single node. If set, this value overrides the default
	// OpenStack volume attachment limit.
	// +optional
	NodeVolumeAttachLimit *uint `json:"nodeVolumeAttachLimit,omitempty"`
}

// NOOP.
type PacketCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	APIKey       string `json:"apiKey,omitempty"`
	ProjectID    string `json:"projectID,omitempty"`
	BillingCycle string `json:"billingCycle"`
}

// GCPCloudSpec specifies access data to GCP.
type GCPCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// The Google Service Account (JSON format), encoded with base64.
	ServiceAccount string `json:"serviceAccount,omitempty"`
	Network        string `json:"network"`
	Subnetwork     string `json:"subnetwork"`
	// A CIDR range that will be used to allow access to the node port range in the firewall rules to.
	// If NodePortsAllowedIPRange nor NodePortsAllowedIPRanges is set, the node port range can be accessed from anywhere.
	NodePortsAllowedIPRange string `json:"nodePortsAllowedIPRange,omitempty"`
	// Optional: CIDR ranges that will be used to allow access to the node port range in the firewall rules to.
	// If NodePortsAllowedIPRange nor NodePortsAllowedIPRanges is set,  the node port range can be accessed from anywhere.
	NodePortsAllowedIPRanges *NetworkRanges `json:"nodePortsAllowedIPRanges,omitempty"`
}

// KubevirtCloudSpec specifies the access data to Kubevirt.
type KubevirtCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// The cluster's kubeconfig file, encoded with base64.
	Kubeconfig    string `json:"kubeconfig,omitempty"`
	CSIKubeconfig string `json:"csiKubeconfig,omitempty"`
	// Custom Images are a good example of this use case.
	PreAllocatedDataVolumes []PreAllocatedDataVolume `json:"preAllocatedDataVolumes,omitempty"`
	// Deprecated: in favor of StorageClasses.
	// InfraStorageClasses is a list of storage classes from KubeVirt infra cluster that are used for
	// initialization of user cluster storage classes by the CSI driver kubevirt (hot pluggable disks)
	InfraStorageClasses []string `json:"infraStorageClasses,omitempty"`
	// StorageClasses is a list of storage classes from KubeVirt infra cluster that are used for
	// initialization of user cluster storage classes by the CSI driver kubevirt (hot pluggable disks.
	// It contains also some flag specifying which one is the default one.
	StorageClasses []KubeVirtInfraStorageClass `json:"storageClasses,omitempty"`
	// ImageCloningEnabled flag enable/disable cloning for a cluster.
	ImageCloningEnabled bool `json:"imageCloningEnabled,omitempty"`
	// VPCName  is a virtual network name dedicated to a single tenant within a KubeVirt.
	VPCName string `json:"vpcName,omitempty"`
	// SubnetName is the name of a subnet that is smaller, segmented portion of a larger network, like a Virtual Private Cloud (VPC).
	SubnetName string `json:"subnetName,omitempty"`
	// CSIDriverOperator configures the kubevirt csi driver operator.
	CSIDriverOperator *KubeVirtCSIDriverOperator `json:"csiDriverOperator,omitempty"`
}

// KubeVirtCSIDriverOperator contains the different configurations for the kubevirt csi driver operator in the user cluster.
type KubeVirtCSIDriverOperator struct {
	// OverwriteRegistry overwrite the images registry that the operator pulls.
	OverwriteRegistry string `json:"overwriteRegistry,omitempty"`
}

type PreAllocatedDataVolume struct {
	Name         string            `json:"name"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	URL          string            `json:"url"`
	Size         string            `json:"size"`
	StorageClass string            `json:"storageClass"`
}

// AlibabaCloudSpec specifies the access data to Alibaba.
type AlibabaCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// The Access Key ID used to authenticate against Alibaba.
	AccessKeyID string `json:"accessKeyID,omitempty"`
	// The Access Key Secret used to authenticate against Alibaba.
	AccessKeySecret string `json:"accessKeySecret,omitempty"`
}

// AnexiaCloudSpec specifies the access data to Anexia.
type AnexiaCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// Token is used to authenticate with the Anexia API.
	Token string `json:"token,omitempty"`
}

// NutanixCSIConfig contains credentials and the endpoint for the Nutanix Prism Element to which the CSI driver connects.
type NutanixCSIConfig struct {
	// Prism Element Username for CSI driver.
	Username string `json:"username,omitempty"`

	// Prism Element Password for CSI driver.
	Password string `json:"password,omitempty"`

	// Prism Element Endpoint to access Nutanix Prism Element for CSI driver.
	Endpoint string `json:"endpoint"`

	// Optional: Port to use when connecting to the Nutanix Prism Element endpoint (defaults to 9440).
	// +optional
	Port *int32 `json:"port,omitempty"`

	// Storage Class options

	// Optional: defaults to "SelfServiceContainer".
	// +optional
	StorageContainer string `json:"storageContainer,omitempty"`

	// Optional: defaults to "xfs"
	// +optional
	Fstype string `json:"fstype,omitempty"`

	// Optional: defaults to "false".
	// +optional
	SsSegmentedIscsiNetwork *bool `json:"ssSegmentedIscsiNetwork,omitempty"`
}

// NutanixCloudSpec specifies the access data to Nutanix.
type NutanixCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// ClusterName is the Nutanix cluster that this user cluster will be deployed to.
	ClusterName string `json:"clusterName"`

	// The name of the project that this cluster is deployed into. If none is given, no project will be used.
	// +optional
	ProjectName string `json:"projectName,omitempty"`

	// Optional: Used to configure a HTTP proxy to access Nutanix Prism Central.
	ProxyURL string `json:"proxyURL,omitempty"`
	// Username to access the Nutanix Prism Central API.
	Username string `json:"username,omitempty"`
	// Password corresponding to the provided user.
	Password string `json:"password,omitempty"`

	// NutanixCSIConfig for CSI driver that connects to a prism element.
	// +optional
	CSI *NutanixCSIConfig `json:"csi,omitempty"`
}

// +kubebuilder:validation:Enum=HealthStatusDown;HealthStatusUp;HealthStatusProvisioning

type HealthStatus string

const (
	HealthStatusDown         = HealthStatus("HealthStatusDown")
	HealthStatusUp           = HealthStatus("HealthStatusUp")
	HealthStatusProvisioning = HealthStatus("HealthStatusProvisioning")
)

// ExtendedClusterHealth stores health information of a cluster.
type ExtendedClusterHealth struct {
	Apiserver         HealthStatus `json:"apiserver,omitempty"`
	Scheduler         HealthStatus `json:"scheduler,omitempty"`
	Controller        HealthStatus `json:"controller,omitempty"`
	MachineController HealthStatus `json:"machineController,omitempty"`
	Etcd              HealthStatus `json:"etcd,omitempty"`
	//  Deprecated: OpenVPN will be removed entirely in the future.
	OpenVPN                      HealthStatus  `json:"openvpn,omitempty"`
	Konnectivity                 HealthStatus  `json:"konnectivity,omitempty"`
	CloudProviderInfrastructure  HealthStatus  `json:"cloudProviderInfrastructure,omitempty"`
	UserClusterControllerManager HealthStatus  `json:"userClusterControllerManager,omitempty"`
	ApplicationController        HealthStatus  `json:"applicationController,omitempty"`
	GatekeeperController         *HealthStatus `json:"gatekeeperController,omitempty"`
	GatekeeperAudit              *HealthStatus `json:"gatekeeperAudit,omitempty"`
	Monitoring                   *HealthStatus `json:"monitoring,omitempty"`
	Logging                      *HealthStatus `json:"logging,omitempty"`
	AlertmanagerConfig           *HealthStatus `json:"alertmanagerConfig,omitempty"`
	MLAGateway                   *HealthStatus `json:"mlaGateway,omitempty"`
	OperatingSystemManager       *HealthStatus `json:"operatingSystemManager,omitempty"`
	KubernetesDashboard          *HealthStatus `json:"kubernetesDashboard,omitempty"`
	KubeLB                       *HealthStatus `json:"kubelb,omitempty"`
	Kyverno                      *HealthStatus `json:"kyverno,omitempty"`
}

// ControlPlaneHealthy returns if all Kubernetes control plane components are healthy.
func (h *ExtendedClusterHealth) ControlPlaneHealthy() bool {
	return h.Etcd == HealthStatusUp &&
		h.Controller == HealthStatusUp &&
		h.Apiserver == HealthStatusUp &&
		h.Scheduler == HealthStatusUp
}

// AllHealthy returns true if all components are healthy. Gatekeeper components not included as they are optional and not
// crucial for cluster functioning.
func (h *ExtendedClusterHealth) AllHealthy() bool {
	return h.ControlPlaneHealthy() &&
		// MachineController is not deployed/supported on Edge clusters and the health status is empty. For all the other
		// providers, the health status is set to "down" when Cluster Health is initialized so it's never empty.
		(h.MachineController == HealthStatusUp || h.MachineController == "") &&
		h.CloudProviderInfrastructure == HealthStatusUp &&
		h.UserClusterControllerManager == HealthStatusUp
}

// ApplicationControllerHealthy checks for health of all essential components and the ApplicationController.
func (h *ExtendedClusterHealth) ApplicationControllerHealthy() bool {
	return h.AllHealthy() &&
		h.ApplicationController == HealthStatusUp
}

type Bytes []byte

// MarshalJSON adds base64 json encoding to the Bytes type.
func (bs Bytes) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", base64.StdEncoding.EncodeToString(bs))), nil
}

// UnmarshalJSON adds base64 json decoding to the Bytes type.
func (bs *Bytes) UnmarshalJSON(src []byte) error {
	if len(src) < 2 {
		return errors.New("base64 string expected")
	}
	if src[0] != '"' || src[len(src)-1] != '"' {
		return errors.New("\" quotations expected")
	}
	if len(src) == 2 {
		*bs = nil
		return nil
	}
	var err error
	*bs, err = base64.StdEncoding.DecodeString(string(src[1 : len(src)-1]))
	return err
}

// Base64 converts a Bytes instance to a base64 string.
func (bs Bytes) Base64() string {
	if []byte(bs) == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(bs))
}

// NewBytes creates a Bytes instance from a base64 string, returning nil for an empty base64 string.
func NewBytes(b64 string) Bytes {
	if b64 == "" {
		return Bytes(nil)
	}
	bs, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(fmt.Sprintf("Invalid base64 string %q", b64))
	}
	return Bytes(bs)
}

func (c *Cluster) GetSecretName() string {
	// new clusters might not have a name yet (if the user used GenerateName),
	// so we must be careful when constructing the Secret name
	clusterName := c.Name
	if clusterName == "" {
		clusterName = rand.String(5)

		if c.GenerateName != "" {
			clusterName = fmt.Sprintf("%s-%s", strings.TrimSuffix(c.GenerateName, "-"), clusterName)
		}
	}

	if c.Spec.Cloud.AWS != nil {
		return fmt.Sprintf("%s-aws-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.Azure != nil {
		return fmt.Sprintf("%s-azure-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.Baremetal != nil {
		return fmt.Sprintf("%s-baremetal-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.Digitalocean != nil {
		return fmt.Sprintf("%s-digitalocean-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.GCP != nil {
		return fmt.Sprintf("%s-gcp-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.Hetzner != nil {
		return fmt.Sprintf("%s-hetzner-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.Openstack != nil {
		return fmt.Sprintf("%s-openstack-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.Kubevirt != nil {
		return fmt.Sprintf("%s-kubevirt-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.VSphere != nil {
		return fmt.Sprintf("%s-vsphere-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.Alibaba != nil {
		return fmt.Sprintf("%s-alibaba-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.Anexia != nil {
		return fmt.Sprintf("%s-anexia-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.Nutanix != nil {
		return fmt.Sprintf("%s-nutanix-%s", CredentialPrefix, clusterName)
	}
	if c.Spec.Cloud.VMwareCloudDirector != nil {
		return fmt.Sprintf("%s-vmware-cloud-director-%s", CredentialPrefix, clusterName)
	}
	return ""
}

// IsEncryptionConfigurationEnabled returns whether encryption-at-rest is configured on this cluster.
func (c *Cluster) IsEncryptionEnabled() bool {
	return c.Spec.Features[ClusterFeatureEncryptionAtRest] && c.Spec.EncryptionConfiguration != nil && c.Spec.EncryptionConfiguration.Enabled
}

// IsEncryptionActive returns whether encryption-at-rest is active on this cluster. This can still be
// the case when encryption configuration has been disabled, as encrypted resources require a decryption.
func (c *Cluster) IsEncryptionActive() bool {
	return c.Status.HasConditionValue(ClusterConditionEncryptionInitialized, corev1.ConditionTrue)
}
