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
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const GlobalSettingsName = "globalsettings"

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// KubermaticSetting is the type representing a KubermaticSetting.
// These settings affect the KKP dashboard and are not relevant when
// using the Kube API on the master/seed clusters directly.
type KubermaticSetting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SettingSpec `json:"spec,omitempty"`
}

// allowedOperatingSystems defines a map of operating systems that can be used for the machines.
type allowedOperatingSystems map[providerconfig.OperatingSystem]bool

type ClusterBackupOptions struct {
	// DefaultChecksumAlgorithm allows setting a default checksum algorithm used by Velero for uploading objects to S3.
	//
	// Optional
	DefaultChecksumAlgorithm *string `json:"defaultChecksumAlgorithm,omitempty"`
}

type SettingSpec struct {
	// CustomLinks are additional links that can be shown the dashboard's footer.
	CustomLinks CustomLinks `json:"customLinks"`
	// DefaultNodeCount is the default number of replicas for the initial MachineDeployment.
	DefaultNodeCount int8 `json:"defaultNodeCount"`
	// DisplayDemoInfo controls whether a "Demo System" hint is shown in the footer.
	DisplayDemoInfo bool `json:"displayDemoInfo"`
	// DisplayDemoInfo controls whether a a link to the KKP API documentation is shown in the footer.
	DisplayAPIDocs bool `json:"displayAPIDocs"`
	// DisplayDemoInfo controls whether a a link to TOS is shown in the footer.
	DisplayTermsOfService bool `json:"displayTermsOfService"`
	// EnableDashboard enables the link to the Kubernetes dashboard for a user cluster.
	EnableDashboard bool `json:"enableDashboard"`

	// +kubebuilder:default=false

	// EnableWebTerminal enables the Web Terminal feature for the user clusters.
	// Deprecated: EnableWebTerminal is deprecated and should be removed in KKP 2.27+. Please use webTerminalOptions instead. When webTerminalOptions.enabled is set then this field will be ignored.
	EnableWebTerminal bool `json:"enableWebTerminal,omitempty"`

	// +kubebuilder:default=true

	// EnableShareCluster enables the Share Cluster feature for the user clusters.
	EnableShareCluster *bool `json:"enableShareCluster,omitempty"`

	OIDCKubeconfigOptions OIDCKubeconfigOptions `json:"oidcKubeconfigOptions"` //nolint:tagliatelle

	// EnableClusterBackup enables the Cluster Backup feature in the dashboard.
	EnableClusterBackups *bool `json:"enableClusterBackup,omitempty"`

	// EnableEtcdBackup enables the etcd Backup feature in the dashboard.
	EnableEtcdBackup bool `json:"enableEtcdBackup,omitempty"`

	// DisableAdminKubeconfig disables the admin kubeconfig functionality on the dashboard.
	DisableAdminKubeconfig bool `json:"disableAdminKubeconfig,omitempty"`

	// UserProjectsLimit is the maximum number of projects a user can create.
	UserProjectsLimit           int64 `json:"userProjectsLimit"`
	RestrictProjectCreation     bool  `json:"restrictProjectCreation"`
	RestrictProjectDeletion     bool  `json:"restrictProjectDeletion"`
	RestrictProjectModification bool  `json:"restrictProjectModification"`

	EnableExternalClusterImport bool `json:"enableExternalClusterImport"`

	// CleanupOptions control what happens when a cluster is deleted via the dashboard.
	// +optional
	CleanupOptions CleanupOptions `json:"cleanupOptions,omitempty"`
	// +optional
	OpaOptions OpaOptions `json:"opaOptions,omitempty"`
	// +optional
	MlaOptions MlaOptions `json:"mlaOptions,omitempty"`

	MlaAlertmanagerPrefix string `json:"mlaAlertmanagerPrefix"`
	MlaGrafanaPrefix      string `json:"mlaGrafanaPrefix"`

	// Notifications are the configuration for notifications on dashboard.
	// +optional
	Notifications NotificationsOptions `json:"notifications,omitempty"`

	// ProviderConfiguration are the cloud provider specific configurations on dashboard.
	// +optional
	ProviderConfiguration ProviderConfiguration `json:"providerConfiguration,omitempty"`

	// WebTerminalOptions are the configurations for the Web Terminal feature.
	// +optional
	WebTerminalOptions *WebTerminalOptions `json:"webTerminalOptions,omitempty"`

	// MachineDeploymentVMResourceQuota is used to filter out allowed machine flavors based on the specified resource limits like CPU, Memory, and GPU etc.
	MachineDeploymentVMResourceQuota *MachineFlavorFilter `json:"machineDeploymentVMResourceQuota,omitempty"`

	// AllowedOperatingSystems shows if the operating system is allowed to be use in the machinedeployment.
	AllowedOperatingSystems allowedOperatingSystems `json:"allowedOperatingSystems,omitempty"`

	// DefaultProjectResourceQuota allows to configure a default project resource quota which
	// will be set for all projects that do not have a custom quota already set. EE-version only.
	// +optional
	DefaultProjectResourceQuota *DefaultProjectResourceQuota `json:"defaultQuota,omitempty"`

	// +optional
	MachineDeploymentOptions MachineDeploymentOptions `json:"machineDeploymentOptions,omitempty"`

	// DisableChangelogPopup disables the changelog popup in KKP dashboard.
	DisableChangelogPopup bool `json:"disableChangelogPopup,omitempty"`

	// StaticLabels are a list of labels that can be used for the clusters.
	StaticLabels []StaticLabel `json:"staticLabels,omitempty"`

	// Annotations are the settings for the annotations in KKP UI.
	Annotations AnnotationSettings `json:"annotations,omitempty"`

	// The announcement feature allows administrators to broadcast important messages to all users.
	Announcements map[string]Announcement `json:"announcements,omitempty"`

	ClusterBackupOptions *ClusterBackupOptions `json:"clusterBackupOptions,omitempty"`

	// TODO: Datacenters, presets, user management, Google Analytics and default addons.
}

// AnnotationSettings is the settings for the annotations.
type AnnotationSettings struct {
	// +kubebuilder:default:={"kubectl.kubernetes.io/last-applied-configuration", "kubermatic.io/initial-application-installations-request", "kubermatic.io/initial-machinedeployment-request", "kubermatic.io/initial-cni-values-request"}

	// HiddenAnnotations are the annotations that are hidden from the user in the UI.
	// +optional
	HiddenAnnotations []string `json:"hiddenAnnotations,omitempty"`

	// +kubebuilder:default:={"presetName"}

	// ProtectedAnnotations are the annotations that are visible in the UI but cannot be added or modified by the user.
	// +optional
	ProtectedAnnotations []string `json:"protectedAnnotations,omitempty"`
}

func (s SettingSpec) HasDefaultProjectResourceQuota() bool {
	return s.DefaultProjectResourceQuota != nil && !s.DefaultProjectResourceQuota.Quota.IsEmpty()
}

type CustomLinks []CustomLink

type CustomLink struct {
	Label    string `json:"label"`
	URL      string `json:"url"`
	Icon     string `json:"icon"`
	Location string `json:"location"`
}

// Controls generation of OIDC-based kubeconfigs.
type OIDCKubeconfigOptions struct {
	// Enable OIDC authentication in the generated kubeconfig.
	Enabled bool `json:"enabled"`
	// If true, Generate the kubeconfig in a format compatible with the kubelogin exec plugin.
	KubeLoginCompatibility bool `json:"kubeLoginCompatibility,omitempty"`
}

type CleanupOptions struct {
	// Enable checkboxes that allow the user to ask for LoadBalancers and PVCs
	// to be deleted in order to not leave potentially expensive resources behind.
	Enabled bool `json:"enabled,omitempty"`
	// If enforced is set to true, the cleanup of LoadBalancers and PVCs is
	// enforced.
	Enforced bool `json:"enforced,omitempty"`
}

type OpaOptions struct {
	Enabled  bool `json:"enabled,omitempty"`
	Enforced bool `json:"enforced,omitempty"`
}

type MlaOptions struct {
	LoggingEnabled     bool `json:"loggingEnabled,omitempty"`
	LoggingEnforced    bool `json:"loggingEnforced,omitempty"`
	MonitoringEnabled  bool `json:"monitoringEnabled,omitempty"`
	MonitoringEnforced bool `json:"monitoringEnforced,omitempty"`
}

type NotificationsOptions struct {
	// HideErrors will silence error notifications for the dashboard.
	HideErrors bool `json:"hideErrors,omitempty"`
	// HideErrorEvents will silence error events for the dashboard.
	HideErrorEvents bool `json:"hideErrorEvents,omitempty"`
}

type ProviderConfiguration struct {
	// OpenStack are the configurations for openstack provider.
	OpenStack OpenStack `json:"openStack,omitempty"`

	// VMwareCloudDirector are the configurations for VMware Cloud Director provider.
	VMwareCloudDirector VMwareCloudDirectorSettings `json:"vmwareCloudDirector,omitempty"`
}

type WebTerminalOptions struct {
	// Enabled enables the Web Terminal feature for the user clusters.
	Enabled *bool `json:"enabled,omitempty"`

	// EnableInternetAccess enables the Web Terminal feature to access the internet.
	EnableInternetAccess bool `json:"enableInternetAccess,omitempty"`

	// AdditionalEnvironmentVariables are the additional environment variables that can be set for the Web Terminal.
	AdditionalEnvironmentVariables []corev1.EnvVar `json:"additionalEnvironmentVariables,omitempty"`
}

type OpenStack struct {
	// EnforceCustomDisk will enforce the custom disk option for machines for the dashboard.
	EnforceCustomDisk bool `json:"enforceCustomDisk,omitempty"`
}

// +kubebuilder:validation:Enum=DHCP;POOL
type ipAllocationMode string

type VMwareCloudDirectorSettings struct {
	// IPAllocationModes are the allowed IP allocation modes for the VMware Cloud Director provider. If not set, all modes are allowed.
	IPAllocationModes []ipAllocationMode `json:"ipAllocationModes,omitempty"`
}

type MachineDeploymentOptions struct {
	// AutoUpdatesEnabled enables the auto updates option for machine deployments on the dashboard.
	// In case of flatcar linux, this will enable automatic updates through update engine and for other operating systems,
	// this will enable package updates on boot for the machines.
	AutoUpdatesEnabled bool `json:"autoUpdatesEnabled,omitempty"`
	// AutoUpdatesEnforced enforces the auto updates option for machine deployments on the dashboard.
	// In case of flatcar linux, this will enable automatic updates through update engine and for other operating systems,
	// this will enable package updates on boot for the machines.
	AutoUpdatesEnforced bool `json:"autoUpdatesEnforced,omitempty"`
}

// DefaultProjectResourceQuota contains the default resource quota which will be set for all
// projects that do not have a custom quota already set.
type DefaultProjectResourceQuota struct {
	// Quota specifies the default CPU, Memory and Storage quantities for all the projects.
	Quota ResourceDetails `json:"quota,omitempty"`
}

// StaticLabel is a label that can be used for the clusters.
type StaticLabel struct {
	Key       string   `json:"key"`
	Values    []string `json:"values"`
	Default   bool     `json:"default"`
	Protected bool     `json:"protected"`
}

// The announcement feature allows administrators to broadcast important messages to all users.
type Announcement struct {
	// The message content of the announcement.
	Message string `json:"message"`
	// Indicates whether the announcement is active.
	IsActive bool `json:"isActive"`
	// Timestamp when the announcement was created.
	CreatedAt metav1.Time `json:"createdAt"`
	// Expiration date for the announcement.
	// +optional
	Expires *metav1.Time `json:"expires,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// KubermaticSettingList is a list of settings.
type KubermaticSettingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []KubermaticSetting `json:"items"`
}
