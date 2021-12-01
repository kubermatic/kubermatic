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

package v2

import (
	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	crdapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
)

// ConstraintTemplate represents a gatekeeper ConstraintTemplate
// swagger:model ConstraintTemplate
type ConstraintTemplate struct {
	Name string `json:"name"`

	Spec   crdapiv1.ConstraintTemplateSpec  `json:"spec"`
	Status v1beta1.ConstraintTemplateStatus `json:"status"`
}

// Constraint represents a gatekeeper Constraint
// swagger:model Constraint
type Constraint struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`

	Spec   crdapiv1.ConstraintSpec `json:"spec"`
	Status *ConstraintStatus       `json:"status,omitempty"`
}

// ConstraintStatus represents a constraint status which holds audit info
type ConstraintStatus struct {
	Enforcement    string      `json:"enforcement,omitempty"`
	AuditTimestamp string      `json:"auditTimestamp,omitempty"`
	Violations     []Violation `json:"violations,omitempty"`
	Synced         *bool       `json:"synced,omitempty"`
}

// Violation represents a gatekeeper constraint violation
type Violation struct {
	EnforcementAction string `json:"enforcementAction,omitempty"`
	Kind              string `json:"kind,omitempty"`
	Message           string `json:"message,omitempty"`
	Name              string `json:"name,omitempty"`
	Namespace         string `json:"namespace,omitempty"`
}

// GatekeeperConfig represents a gatekeeper config
// swagger:model GatekeeperConfig
type GatekeeperConfig struct {
	Spec GatekeeperConfigSpec `json:"spec"`
}

type GatekeeperConfigSpec struct {
	// Configuration for syncing k8s objects
	Sync Sync `json:"sync,omitempty"`

	// Configuration for validation
	Validation Validation `json:"validation,omitempty"`

	// Configuration for namespace exclusion
	Match []MatchEntry `json:"match,omitempty"`

	// Configuration for readiness tracker
	Readiness ReadinessSpec `json:"readiness,omitempty"`
}

type Sync struct {
	// If non-empty, entries on this list will be replicated into OPA
	SyncOnly []GVK `json:"syncOnly,omitempty"`
}

type Validation struct {
	// List of requests to trace. Both "user" and "kinds" must be specified
	Traces []Trace `json:"traces,omitempty"`
}

type Trace struct {
	// Only trace requests from the specified user
	User string `json:"user,omitempty"`
	// Only trace requests of the following GroupVersionKind
	Kind GVK `json:"kind,omitempty"`
	// Also dump the state of OPA with the trace. Set to `All` to dump everything.
	Dump string `json:"dump,omitempty"`
}

type MatchEntry struct {
	// Namespaces which will be excluded
	ExcludedNamespaces []string `json:"excludedNamespaces,omitempty"`
	// Processes which will be excluded in the given namespaces (sync, webhook, audit, *)
	Processes []string `json:"processes,omitempty"`
}

type ReadinessSpec struct {
	// enables stats for gatekeeper audit
	StatsEnabled bool `json:"statsEnabled,omitempty"`
}

// GVK group version kind of a resource
type GVK struct {
	Group   string `json:"group,omitempty"`
	Version string `json:"version,omitempty"`
	Kind    string `json:"kind,omitempty"`
}

// PresetList represents a list of presets
// swagger:model PresetList
type PresetList struct {
	Items []Preset `json:"items"`
}

// Preset represents a preset
// swagger:model Preset
type Preset struct {
	Name      string           `json:"name"`
	Enabled   bool             `json:"enabled"`
	Providers []PresetProvider `json:"providers"`
}

// PresetProvider represents a preset provider
// swagger:model PresetProvider
type PresetProvider struct {
	Name    crdapiv1.ProviderType `json:"name"`
	Enabled bool                  `json:"enabled"`
}

// Alertmanager represents an Alertmanager Configuration
// swagger:model Alertmanager
type Alertmanager struct {
	Spec AlertmanagerSpec `json:"spec"`
}

type AlertmanagerSpec struct {
	// Config contains the alertmanager configuration in YAML
	Config []byte `json:"config"`
}

// SeedSettings represents settings for a Seed cluster
// swagger:model SeedSettings
type SeedSettings struct {
	// the Seed level MLA (Monitoring, Logging, and Alerting) stack settings
	MLA MLA `json:"mla"`
	// the Seed level metering settings
	Metering crdapiv1.MeteringConfigurations `json:"metering"`
	// the Seed level seed dns overwrite
	SeedDNSOverwrite string `json:"seedDNSOverwrite,omitempty"`
}

type MLA struct {
	// whether the user cluster MLA (Monitoring, Logging & Alerting) stack is enabled in the seed
	UserClusterMLAEnabled bool `json:"user_cluster_mla_enabled"`
}

// ClusterTemplate represents a ClusterTemplate object
// swagger:model ClusterTemplate
type ClusterTemplate struct {
	Name string `json:"name"`
	ID   string `json:"id"`

	ProjectID      string                  `json:"projectID,omitempty"`
	User           string                  `json:"user,omitempty"`
	Scope          string                  `json:"scope"`
	UserSSHKeys    []ClusterTemplateSSHKey `json:"userSshKeys,omitempty"`
	Cluster        *apiv1.Cluster          `json:"cluster,omitempty"`
	NodeDeployment *apiv1.NodeDeployment   `json:"nodeDeployment,omitempty"`
}

// ClusterTemplateSSHKey represents SSH Key object for Cluster Template
// swagger:model ClusterTemplateSSHKey
type ClusterTemplateSSHKey struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// ClusterTemplateList represents a ClusterTemplate list
// swagger:model ClusterTemplateList
type ClusterTemplateList []ClusterTemplate

// ClusterTemplateInstance represents a ClusterTemplateInstance object
// swagger:model ClusterTemplateInstance
type ClusterTemplateInstance struct {
	Name string `json:"name"`

	Spec crdapiv1.ClusterTemplateInstanceSpec `json:"spec"`
}

// RuleGroup represents a rule group of recording and alerting rules.
// swagger:model RuleGroup
type RuleGroup struct {
	// contains the RuleGroup data. Ref: https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#rule_group
	Data []byte `json:"data"`
	// the type of this ruleGroup applies to. It can be `Metrics`.
	Type crdapiv1.RuleGroupType `json:"type"`
}

// AllowedRegistry represents a object containing a allowed image registry prefix
// swagger:model AllowedRegistry
type AllowedRegistry struct {
	Name string `json:"name"`

	Spec crdapiv1.AllowedRegistrySpec `json:"spec"`
}

// EtcdBackupConfig represents an object holding the configuration for etcd backups
// swagger:model EtcdBackupConfig
type EtcdBackupConfig struct {
	apiv1.ObjectMeta

	Spec   EtcdBackupConfigSpec   `json:"spec"`
	Status EtcdBackupConfigStatus `json:"status"`
}

type EtcdBackupConfigStatus struct {
	// CurrentBackups tracks the creation and deletion progress if all backups managed by the EtcdBackupConfig
	CurrentBackups []BackupStatus `json:"lastBackups,omitempty"`
	// Conditions contains conditions of the EtcdBackupConfig
	Conditions []EtcdBackupConfigCondition `json:"conditions,omitempty"`
	// If the controller was configured with a cleanupContainer, CleanupRunning keeps track of the corresponding job
	CleanupRunning bool `json:"cleanupRunning,omitempty"`
}

type BackupStatus struct {
	// ScheduledTime will always be set when the BackupStatus is created, so it'll never be nil
	ScheduledTime      *apiv1.Time                `json:"scheduledTime,omitempty"`
	BackupName         string                     `json:"backupName,omitempty"`
	JobName            string                     `json:"jobName,omitempty"`
	BackupStartTime    *apiv1.Time                `json:"backupStartTime,omitempty"`
	BackupFinishedTime *apiv1.Time                `json:"backupFinishedTime,omitempty"`
	BackupPhase        crdapiv1.BackupStatusPhase `json:"backupPhase,omitempty"`
	BackupMessage      string                     `json:"backupMessage,omitempty"`
	DeleteJobName      string                     `json:"deleteJobName,omitempty"`
	DeleteStartTime    *apiv1.Time                `json:"deleteStartTime,omitempty"`
	DeleteFinishedTime *apiv1.Time                `json:"deleteFinishedTime,omitempty"`
	DeletePhase        crdapiv1.BackupStatusPhase `json:"deletePhase,omitempty"`
	DeleteMessage      string                     `json:"deleteMessage,omitempty"`
}

type EtcdBackupConfigCondition struct {
	// Type of EtcdBackupConfig condition.
	Type crdapiv1.EtcdBackupConfigConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime apiv1.Time `json:"lastHeartbeatTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime apiv1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// EtcdBackupConfigSpec represents an object holding the etcd backup configuration specification
// swagger:model EtcdBackupConfigSpec
type EtcdBackupConfigSpec struct {
	// ClusterID is the id of the cluster which will be backed up
	ClusterID string `json:"clusterId"`
	// Schedule is a cron expression defining when to perform
	// the backup. If not set, the backup is performed exactly
	// once, immediately.
	Schedule string `json:"schedule,omitempty"`
	// Keep is the number of backups to keep around before deleting the oldest one
	// If not set, defaults to DefaultKeptBackupsCount. Only used if Schedule is set.
	Keep *int `json:"keep,omitempty"`
	// Destination indicates where the backup will be stored. The destination name should correspond to a destination in
	// the cluster's Seed.Spec.EtcdBackupRestore. If empty, it will use the legacy destination in Seed.Spec.BackupRestore
	Destination string `json:"destination,omitempty"`
}

// EtcdRestore represents an object holding the configuration for etcd backup restore
// swagger:model EtcdRestore
type EtcdRestore struct {
	Name string `json:"name"`

	Spec   EtcdRestoreSpec   `json:"spec"`
	Status EtcdRestoreStatus `json:"status"`
}

type EtcdRestoreStatus struct {
	Phase       crdapiv1.EtcdRestorePhase `json:"phase"`
	RestoreTime *apiv1.Time               `json:"restoreTime,omitempty"`
}

// EtcdRestoreSpec represents an object holding the etcd backup restore configuration specification
// swagger:model EtcdRestoreSpec
type EtcdRestoreSpec struct {
	// ClusterID is the id of the cluster which will be restored from the backup
	ClusterID string `json:"clusterId"`
	// BackupName is the name of the backup to restore from
	BackupName string `json:"backupName"`
	// BackupDownloadCredentialsSecret is the name of a secret in the cluster-xxx namespace containing
	// credentials needed to download the backup
	BackupDownloadCredentialsSecret string `json:"backupDownloadCredentialsSecret,omitempty"`
}

// OIDCSpec contains OIDC params that can be used to access user cluster.
// swagger:model OIDCSpec
type OIDCSpec struct {
	IssuerURL    string `json:"issuerUrl,omitempty"`
	ClientID     string `json:"clientId,omitempty"`
	ClientSecret string `json:"clientSecret,omitempty"`
}

// BackupCredentials contains credentials for etcd backups
// swagger:model BackupCredentials
type BackupCredentials struct {
	// S3BackupCredentials holds credentials for a S3 client compatible backup destination
	S3BackupCredentials S3BackupCredentials `json:"s3,omitempty"`
	// Destination corresponds to the Seeds Seed.Spec.EtcdBackupRestore.Destinations, it defines for which destination
	// the backup credentials will be created. If set, it updates the credentials ref in the related Seed BackupDestination
	Destination string `json:"destination,omitempty"`
}

// S3BackupCredentials contains credentials for S3 etcd backups
// swagger:model S3BackupCredentials
type S3BackupCredentials struct {
	AccessKeyID     string `json:"accessKeyId,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
}

// MLAAdminSetting represents an object holding admin setting options for user cluster MLA (Monitoring, Logging and Alerting).
// swagger:model MLAAdminSetting
type MLAAdminSetting struct {
	// MonitoringRateLimits contains rate-limiting configuration for monitoring in the user cluster.
	MonitoringRateLimits *crdapiv1.MonitoringRateLimitSettings `json:"monitoringRateLimits,omitempty"`
	// LoggingRateLimits contains rate-limiting configuration logging in the user cluster.
	LoggingRateLimits *crdapiv1.LoggingRateLimitSettings `json:"loggingRateLimits,omitempty"`
}

// ExternalCluster represents an object holding cluster details
// swagger:model ExternalCluster
type ExternalCluster struct {
	apiv1.Cluster `json:",inline"`
	Cloud         *ExternalClusterCloudSpec `json:"cloud,omitempty"`
}

// ExternalClusterCloudSpec represents an object holding cluster cloud details
// swagger:model ExternalClusterCloudSpec
type ExternalClusterCloudSpec struct {
	GKE *GKECloudSpec `json:"gke,omitempty"`
	EKS *EKSCloudSpec `json:"eks,omitempty"`
	AKS *AKSCloudSpec `json:"aks,omitempty"`
}

type GKECloudSpec struct {
	Name           string `json:"name"`
	ServiceAccount string `json:"serviceAccount,omitempty"`
	Zone           string `json:"zone"`
}

type EKSCloudSpec struct {
	Name            string `json:"name"`
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
	Region          string `json:"region"`
}

type AKSCloudSpec struct {
	Name           string `json:"name"`
	TenantID       string `json:"tenantID"`
	SubscriptionID string `json:"subscriptionID"`
	ClientID       string `json:"clientID"`
	ClientSecret   string `json:"clientSecret"`
	ResourceGroup  string `json:"resourceGroup"`
}

// ExternalClusterNode represents an object holding external cluster node
// swagger:model ExternalClusterNode
type ExternalClusterNode struct {
	apiv1.Node `json:",inline"`
}

// ExternalClusterMachineDeployment represents an object holding external cluster machine deployment
// swagger:model ExternalClusterMachineDeployment
type ExternalClusterMachineDeployment struct {
	apiv1.NodeDeployment `json:",inline"`
	Cloud                *ExternalClusterMachineDeploymentCloudSpec `json:"cloud,omitempty"`
}

// GKECluster represents a object of GKE cluster.
// swagger:model GKECluster
type GKECluster struct {
	Name       string `json:"name"`
	IsImported bool   `json:"imported"`
	Zone       string `json:"zone"`
}

// GKEClusterList represents an array of GKE clusters.
// swagger:model GKEClusterList
type GKEClusterList []GKECluster

// GKEImage represents an object of GKE image.
// swagger:model GKEImage
type GKEImage struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"default"`
}

// GKEImageList represents an array of GKE images.
// swagger:model GKEImageList
type GKEImageList []GKEImage

// GKEZone represents a object of GKE zone.
// swagger:model GKEZone
type GKEZone struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"default"`
}

// GKEZoneList represents an array of GKE zones.
// swagger:model GKEZoneList
type GKEZoneList []GKEZone

// EKSCluster represents a object of EKS cluster.
// swagger:model EKSCluster
type EKSCluster struct {
	Name       string `json:"name"`
	Region     string `json:"region"`
	IsImported bool   `json:"imported"`
}

// EKSClusterList represents an list of EKS clusters.
// swagger:model EKSClusterList
type EKSClusterList []EKSCluster

// Regions represents an list of AWS regions.
// swagger:model Regions
type Regions []string

// AKSCluster represents a object of AKS cluster.
// swagger:model AKSCluster
type AKSCluster struct {
	Name       string `json:"name"`
	IsImported bool   `json:"imported"`
}

// AKSClusterList represents an list of AKS clusters.
// swagger:model AKSClusterList
type AKSClusterList []AKSCluster

// FeatureGates represents an object holding feature gate settings
// swagger:model FeatureGates
type FeatureGates struct {
	KonnectivityService *bool `json:"konnectivityService,omitempty"`
}

// ExternalClusterMachineDeploymentCloudSpec represents an object holding machine deployment cloud details.
// swagger:model ExternalClusterMachineDeploymentCloudSpec
type ExternalClusterMachineDeploymentCloudSpec struct {
	GKE *GKEMachineDeploymentCloudSpec `json:"gke,omitempty"`
}

// GKEMachineDeploymentCloudSpec represents an object holding GKE machine deployment cloud details.
type GKEMachineDeploymentCloudSpec struct {
	// Autoscaling: Autoscaler configuration for this NodePool. Autoscaler
	// is enabled only if a valid configuration is present.
	Autoscaling *GKENodePoolAutoscaling `json:"autoscaling,omitempty"`

	// Config: The node configuration of the pool.
	Config *GKENodeConfig `json:"config,omitempty"`

	// Management: NodeManagement configuration for this NodePool.
	Management *GKENodeManagement `json:"management,omitempty"`

	// Locations: The list of Google Compute Engine zones
	// (https://cloud.google.com/compute/docs/zones#available) in which the
	// NodePool's nodes should be located. If this value is unspecified
	// during node pool creation, the Cluster.Locations
	// (https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters#Cluster.FIELDS.locations)
	// value will be used, instead. Warning: changing node pool locations
	// will result in nodes being added and/or removed.
	Locations []string `json:"locations,omitempty"`
}

// GKENodeManagement defines the set of node management
// services turned on for the node pool.
type GKENodeManagement struct {
	// AutoRepair: A flag that specifies whether the node auto-repair is
	// enabled for the node pool. If enabled, the nodes in this node pool
	// will be monitored and, if they fail health checks too many times, an
	// automatic repair action will be triggered.
	AutoRepair bool `json:"autoRepair,omitempty"`

	// AutoUpgrade: A flag that specifies whether node auto-upgrade is
	// enabled for the node pool. If enabled, node auto-upgrade helps keep
	// the nodes in your node pool up to date with the latest release
	// version of Kubernetes.
	AutoUpgrade bool `json:"autoUpgrade,omitempty"`
}

// GKENodeConfig Parameters that describe the nodes in a cluster.
type GKENodeConfig struct {
	// DiskSizeGb: Size of the disk attached to each node, specified in GB.
	// The smallest allowed disk size is 10GB. If unspecified, the default
	// disk size is 100GB.
	DiskSizeGb int64 `json:"diskSizeGb,omitempty"`

	// DiskType: Type of the disk attached to each node (e.g. 'pd-standard',
	// 'pd-ssd' or 'pd-balanced') If unspecified, the default disk type is
	// 'pd-standard'
	DiskType string `json:"diskType,omitempty"`

	// ImageType: The image type to use for this node. Note that for a given
	// image type, the latest version of it will be used.
	ImageType string `json:"imageType,omitempty"`

	// LocalSsdCount: The number of local SSD disks to be attached to the
	// node. The limit for this value is dependent upon the maximum number
	// of disks available on a machine per zone. See:
	// https://cloud.google.com/compute/docs/disks/local-ssd for more
	// information.
	LocalSsdCount int64 `json:"localSsdCount,omitempty"`

	// MachineType: The name of a Google Compute Engine machine type
	// (https://cloud.google.com/compute/docs/machine-types) If unspecified,
	// the default machine type is `e2-medium`.
	MachineType string `json:"machineType,omitempty"`

	// Labels: The map of Kubernetes labels (key/value pairs) to be applied
	// to each node. These will added in addition to any default label(s)
	// that Kubernetes may apply to the node. In case of conflict in label
	// keys, the applied set may differ depending on the Kubernetes version
	// -- it's best to assume the behavior is undefined and conflicts should
	// be avoided. For more information, including usage and the valid
	// values, see:
	// https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
	Labels map[string]string `json:"labels,omitempty"`
}

// GKENodePoolAutoscaling contains information
// required by cluster autoscaler to adjust the size of the node pool to
// the current cluster usage.
type GKENodePoolAutoscaling struct {
	// Autoprovisioned: Can this node pool be deleted automatically.
	Autoprovisioned bool `json:"autoprovisioned,omitempty"`

	// Enabled: Is autoscaling enabled for this node pool.
	Enabled bool `json:"enabled,omitempty"`

	// MaxNodeCount: Maximum number of nodes in the NodePool. Must be >=
	// min_node_count. There has to enough quota to scale up the cluster.
	MaxNodeCount int64 `json:"maxNodeCount,omitempty"`

	// MinNodeCount: Minimum number of nodes in the NodePool. Must be >= 1
	// and <= max_node_count.
	MinNodeCount int64 `json:"minNodeCount,omitempty"`
}
