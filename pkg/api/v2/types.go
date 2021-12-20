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
	kubevirtv1 "kubevirt.io/api/core/v1"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	crdapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// Destination indicates where the backup was stored. The destination name should correspond to a destination in
	// the cluster's Seed.Spec.EtcdBackupRestore. If empty, it will use the legacy destination configured in Seed.Spec.BackupRestore
	Destination string `json:"destination,omitempty"`
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
	apiv1.ObjectMeta `json:",inline"`
	Labels           map[string]string         `json:"labels,omitempty"`
	Spec             ExternalClusterSpec       `json:"spec"`
	Cloud            *ExternalClusterCloudSpec `json:"cloud,omitempty"`
	Status           ExternalClusterStatus     `json:"status"`
}

type ExternalClusterState string

const (
	// PROVISIONING state indicates the cluster is being created.
	PROVISIONING ExternalClusterState = "PROVISIONING"

	// RUNNING state indicates the cluster has been created and is fully usable.
	RUNNING ExternalClusterState = "RUNNING"

	// RECONCILING state indicates that some work is actively being done on the cluster, such as upgrading the master or
	// node software. Details can be found in the `StatusMessage` field.
	RECONCILING ExternalClusterState = "RECONCILING"

	// DELETING state indicates the cluster is being deleted.
	DELETING ExternalClusterState = "DELETING"

	// UNKNOWN Not set.
	UNKNOWN ExternalClusterState = "UNKNOWN"

	// ERROR state indicates the cluster is unusable. It will be automatically deleted. Details can be found in the
	// `statusMessage` field.
	ERROR ExternalClusterState = "ERROR"
)

// ExternalClusterStatus defines the external cluster status
type ExternalClusterStatus struct {
	State         ExternalClusterState `json:"state"`
	StatusMessage string               `json:"statusMessage"`
}

// ExternalClusterSpec defines the external cluster specification
type ExternalClusterSpec struct {
	// Version desired version of the kubernetes master components
	Version ksemver.Semver `json:"version"`
}

// ExternalClusterCloudSpec represents an object holding cluster cloud details
// swagger:model ExternalClusterCloudSpec
type ExternalClusterCloudSpec struct {
	GKE *GKECloudSpec `json:"gke,omitempty"`
	EKS *EKSCloudSpec `json:"eks,omitempty"`
	AKS *AKSCloudSpec `json:"aks,omitempty"`
}

type GKECloudSpec struct {
	Name           string          `json:"name"`
	ServiceAccount string          `json:"serviceAccount,omitempty"`
	Zone           string          `json:"zone"`
	ClusterSpec    *GKEClusterSpec `json:"clusterSpec,omitempty"`
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
	Name          string `json:"name"`
	ResourceGroup string `json:"resourceGroup"`
	IsImported    bool   `json:"imported"`
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
	AKS *AKSMachineDeploymentCloudSpec `json:"aks,omitempty"`
}

type AKSMachineDeploymentCloudSpec struct {
	// Basics - Settings for creating the agentpool
	Basics *AgentPoolBasics `json:"basics,omitempty"`
	// OptionalSettings - Optional Settings for creating the agentpool
	OptionalSettings *AgentPoolOptionalSettings `json:"optionalSettings,omitempty"`
	// Tags - The tags to be persisted on the agent pool virtual machine scale set.
	Tags map[string]*string `json:"tags"`
}

type AgentPoolBasics struct {
	// Mode - Possible values include: 'System', 'User'
	Mode string `json:"mode,omitempty"`
	// OsType - Possible values include: 'Linux', 'Windows'
	OsType string `json:"osType,omitempty"`
	// OrchestratorVersion - As a best practice, you should upgrade all node pools in an AKS cluster to the same Kubernetes version. The node pool version must have the same major version as the control plane. The node pool minor version must be within two minor versions of the control plane version. The node pool version cannot be greater than the control plane version. For more information see [upgrading a node pool](https://docs.microsoft.com/azure/aks/use-multiple-node-pools#upgrade-a-node-pool).
	OrchestratorVersion *string `json:"orchestratorVersion,omitempty"`
	// AvailabilityZones - The list of Availability zones to use for nodes. This can only be specified if the AgentPoolType property is 'VirtualMachineScaleSets'.
	AvailabilityZones *[]string `json:"availabilityZones,omitempty"`
	// VMSize - VM size availability varies by region. If a node contains insufficient compute resources (memory, cpu, etc) pods might fail to run correctly. For more details on restricted VM sizes, see: https://docs.microsoft.com/azure/aks/quotas-skus-regions
	VMSize *string `json:"vmSize,omitempty"`
	// EnableAutoScaling - Whether to enable auto-scaler
	EnableAutoScaling *bool `json:"enableAutoScaling,omitempty"`
	// MaxCount - The maximum number of nodes for auto-scaling
	MaxCount *int32 `json:"maxCount,omitempty"`
	// MinCount - The minimum number of nodes for auto-scaling
	MinCount *int32 `json:"minCount,omitempty"`
	// Count - Number of agents (VMs) to host docker containers. Allowed values must be in the range of 0 to 1000 (inclusive) for user pools and in the range of 1 to 1000 (inclusive) for system pools. The default value is 1.
	Count *int32 `json:"count,omitempty"`
}

type AgentPoolOptionalSettings struct {
	// MaxPods - The maximum number of pods that can run on a node.
	MaxPods *int32 `json:"maxPods,omitempty"`
	// EnableNodePublicIP - Some scenarios may require nodes in a node pool to receive their own dedicated public IP addresses. A common scenario is for gaming workloads, where a console needs to make a direct connection to a cloud virtual machine to minimize hops. For more information see [assigning a public IP per node](https://docs.microsoft.com/azure/aks/use-multiple-node-pools#assign-a-public-ip-per-node-for-your-node-pools). The default is false.
	EnableNodePublicIP *bool `json:"enableNodePublicIP,omitempty"`
	// UpgradeSettings - Settings for upgrading the agentpool
	UpgradeSettings *AgentPoolUpgradeSettings `json:"upgradeSettings,omitempty"`
	// NodeLabels - The node labels to be persisted across all nodes in agent pool.
	NodeLabels map[string]*string `json:"nodeLabels"`
	// NodeTaints - The taints added to new nodes during node pool create and scale. For example, key=value:NoSchedule.
	NodeTaints *[]string `json:"nodeTaints,omitempty"`
}

type AgentPoolUpgradeSettings struct {
	// MaxSurge - This can either be set to an integer (e.g. '5') or a percentage (e.g. '50%'). If a percentage is specified, it is the percentage of the total agent pool size at the time of the upgrade. For percentages, fractional nodes are rounded up. If not specified, the default is 1. For more information, including best practices, see: https://docs.microsoft.com/azure/aks/upgrade-cluster#customize-node-surge-upgrade
	MaxSurge *string `json:"maxSurge,omitempty"`
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

	// Preemptible: Whether the nodes are created as preemptible VM
	// instances. See:
	// https://cloud.google.com/compute/docs/instances/preemptible for more
	// information about preemptible VM instances.
	Preemptible bool `json:"preemptible,omitempty"`
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

// BackupDestinationNames represents an list of backup destination names.
// swagger:model BackupDestinationNames
type BackupDestinationNames []string

// GKEClusterSpec A Google Kubernetes Engine cluster.
type GKEClusterSpec struct {
	// Autopilot: Autopilot configuration for the cluster.
	Autopilot bool `json:"autopilot,omitempty"`

	// GKEClusterAutoscaling: Cluster-level autoscaling configuration.
	Autoscaling *GKEClusterAutoscaling `json:"autoscaling,omitempty"`

	// ClusterIpv4Cidr: The IP address range of the container pods in this
	// cluster, in CIDR
	// (http://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing)
	// notation (e.g. `10.96.0.0/14`). Leave blank to have one automatically
	// chosen or specify a `/14` block in `10.0.0.0/8`.
	ClusterIpv4Cidr string `json:"clusterIpv4Cidr,omitempty"`

	// DefaultMaxPodsConstraint: The default constraint on the maximum
	// number of pods that can be run simultaneously on a node in the node
	// pool of this cluster. Only honored if cluster created with IP Alias
	// support.
	DefaultMaxPodsConstraint *int64 `json:"defaultMaxPodsConstraint,omitempty"`

	// EnableKubernetesAlpha: Kubernetes alpha features are enabled on this
	// cluster. This includes alpha API groups (e.g. v1alpha1) and features
	// that may not be production ready in the kubernetes version of the
	// master and nodes. The cluster has no SLA for uptime and master/node
	// upgrades are disabled. Alpha enabled clusters are automatically
	// deleted thirty days after creation.
	EnableKubernetesAlpha bool `json:"enableKubernetesAlpha,omitempty"`

	// EnableTpu: Enable the ability to use Cloud TPUs in this cluster.
	EnableTpu bool `json:"enableTpu,omitempty"`

	// InitialClusterVersion: The initial Kubernetes version for this
	// cluster. Valid versions are those found in validMasterVersions
	// returned by getServerConfig. The version can be upgraded over time;
	// such upgrades are reflected in currentMasterVersion and
	// currentNodeVersion. Users may specify either explicit versions
	// offered by Kubernetes Engine or version aliases, which have the
	// following behavior: - "latest": picks the highest valid Kubernetes
	// version - "1.X": picks the highest valid patch+gke.N patch in the 1.X
	// version - "1.X.Y": picks the highest valid gke.N patch in the 1.X.Y
	// version - "1.X.Y-gke.N": picks an explicit Kubernetes version -
	// "","-": picks the default Kubernetes version
	InitialClusterVersion string `json:"initialClusterVersion,omitempty"`

	// InitialNodeCount: The number of nodes to create in this cluster. You
	// must ensure that your Compute Engine resource quota
	// (https://cloud.google.com/compute/quotas) is sufficient for this
	// number of instances. You must also have available firewall and routes
	// quota. For requests, this field should only be used in lieu of a
	// "node_pool" object, since this configuration (along with the
	// "node_config") will be used to create a "NodePool" object with an
	// auto-generated name. Do not use this and a node_pool at the same
	// time. This field is deprecated, use node_pool.initial_node_count
	// instead.
	InitialNodeCount int64 `json:"initialNodeCount,omitempty"`

	// Locations: The list of Google Compute Engine zones
	// (https://cloud.google.com/compute/docs/zones#available) in which the
	// cluster's nodes should be located. This field provides a default
	// value if NodePool.Locations
	// (https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters.nodePools#NodePool.FIELDS.locations)
	// are not specified during node pool creation. Warning: changing
	// cluster locations will update the NodePool.Locations
	// (https://cloud.google.com/kubernetes-engine/docs/reference/rest/v1/projects.locations.clusters.nodePools#NodePool.FIELDS.locations)
	// of all node pools and will result in nodes being added and/or
	// removed.
	Locations []string `json:"locations,omitempty"`

	// Network: The name of the Google Compute Engine network
	// (https://cloud.google.com/compute/docs/networks-and-firewalls#networks)
	// to which the cluster is connected. If left unspecified, the `default`
	// network will be used.
	Network string `json:"network,omitempty"`

	// NodeConfig: Parameters used in creating the cluster's nodes. For
	// requests, this field should only be used in lieu of a "node_pool"
	// object, since this configuration (along with the
	// "initial_node_count") will be used to create a "NodePool" object with
	// an auto-generated name. Do not use this and a node_pool at the same
	// time. For responses, this field will be populated with the node
	// configuration of the first node pool. (For configuration of each node
	// pool, see `node_pool.config`) If unspecified, the defaults are used.
	// This field is deprecated, use node_pool.config instead.
	NodeConfig *GKENodeConfig `json:"nodeConfig,omitempty"`

	// Subnetwork: The name of the Google Compute Engine subnetwork
	// (https://cloud.google.com/compute/docs/subnetworks) to which the
	// cluster is connected.
	Subnetwork string `json:"subnetwork,omitempty"`

	// TpuIpv4CidrBlock: [Output only] The IP address range of the Cloud
	// TPUs in this cluster, in CIDR
	// (http://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing)
	// notation (e.g. `1.2.3.4/29`).
	TpuIpv4CidrBlock string `json:"tpuIpv4CidrBlock,omitempty"`

	// VerticalPodAutoscaling: Cluster-level Vertical Pod Autoscaling
	// configuration.
	VerticalPodAutoscaling bool `json:"verticalPodAutoscaling,omitempty"`
}

// GKEClusterAutoscaling contains global, per-cluster
// information required by Cluster Autoscaler to automatically adjust
// the size of the cluster and create/delete node pools based on the
// current needs.
type GKEClusterAutoscaling struct {
	// AutoprovisioningLocations: The list of Google Compute Engine zones
	// (https://cloud.google.com/compute/docs/zones#available) in which the
	// NodePool's nodes can be created by NAP.
	AutoprovisioningLocations []string `json:"autoprovisioningLocations,omitempty"`

	// AutoprovisioningNodePoolDefaults: AutoprovisioningNodePoolDefaults
	// contains defaults for a node pool created by NAP.
	AutoprovisioningNodePoolDefaults *GKEAutoprovisioningNodePoolDefaults `json:"autoprovisioningNodePoolDefaults,omitempty"`

	// EnableNodeAutoprovisioning: Enables automatic node pool creation and
	// deletion.
	EnableNodeAutoprovisioning bool `json:"enableNodeAutoprovisioning,omitempty"`

	// ResourceLimits: Contains global constraints regarding minimum and
	// maximum amount of resources in the cluster.
	ResourceLimits []*GKEResourceLimit `json:"resourceLimits,omitempty"`
}

// GKEResourceLimit Contains information about amount of some resource in
// the cluster. For memory, value should be in GB.
type GKEResourceLimit struct {
	// Maximum: Maximum amount of the resource in the cluster.
	Maximum int64 `json:"maximum,omitempty,string"`

	// Minimum: Minimum amount of the resource in the cluster.
	Minimum int64 `json:"minimum,omitempty,string"`

	// ResourceType: Resource name "cpu", "memory" or gpu-specific string.
	ResourceType string `json:"resourceType,omitempty"`
}

// GKEAutoprovisioningNodePoolDefaults
// contains defaults for a node pool created by NAP.
type GKEAutoprovisioningNodePoolDefaults struct {
	// BootDiskKmsKey: The Customer Managed Encryption Key used to encrypt
	// the boot disk attached to each node in the node pool. This should be
	// of the form
	// projects/[KEY_PROJECT_ID]/locations/[LOCATION]/keyRings/[RING_NAME]/cr
	// yptoKeys/[KEY_NAME]. For more information about protecting resources
	// with Cloud KMS Keys please see:
	// https://cloud.google.com/compute/docs/disks/customer-managed-encryption
	BootDiskKmsKey string `json:"bootDiskKmsKey,omitempty"`

	// DiskSizeGb: Size of the disk attached to each node, specified in GB.
	// The smallest allowed disk size is 10GB. If unspecified, the default
	// disk size is 100GB.
	DiskSizeGb int64 `json:"diskSizeGb,omitempty"`

	// DiskType: Type of the disk attached to each node (e.g. 'pd-standard',
	// 'pd-ssd' or 'pd-balanced') If unspecified, the default disk type is
	// 'pd-standard'
	DiskType string `json:"diskType,omitempty"`

	// Management: Specifies the node management options for NAP created
	// node-pools.
	Management *GKENodeManagement `json:"management,omitempty"`

	// MinCpuPlatform: Minimum CPU platform to be used for NAP created node
	// pools. The instance may be scheduled on the specified or newer CPU
	// platform. Applicable values are the friendly names of CPU platforms,
	// such as minCpuPlatform: Intel Haswell or minCpuPlatform: Intel Sandy
	// Bridge. For more information, read how to specify min CPU platform
	// (https://cloud.google.com/compute/docs/instances/specify-min-cpu-platform)
	// To unset the min cpu platform field pass "automatic" as field value.
	MinCpuPlatform string `json:"minCpuPlatform,omitempty"`

	// OauthScopes: Scopes that are used by NAP when creating node pools.
	OauthScopes []string `json:"oauthScopes,omitempty"`

	// ServiceAccount: The Google Cloud Platform Service Account to be used
	// by the node VMs.
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// ShieldedInstanceConfig: Shielded Instance options.
	ShieldedInstanceConfig *GKEShieldedInstanceConfig `json:"shieldedInstanceConfig,omitempty"`

	// UpgradeSettings: Specifies the upgrade settings for NAP created node
	// pools
	UpgradeSettings *GKEUpgradeSettings `json:"upgradeSettings,omitempty"`
}

// GKEShieldedInstanceConfig a set of Shielded Instance options.
type GKEShieldedInstanceConfig struct {
	// EnableIntegrityMonitoring: Defines whether the instance has integrity
	// monitoring enabled. Enables monitoring and attestation of the boot
	// integrity of the instance. The attestation is performed against the
	// integrity policy baseline. This baseline is initially derived from
	// the implicitly trusted boot image when the instance is created.
	EnableIntegrityMonitoring bool `json:"enableIntegrityMonitoring,omitempty"`

	// EnableSecureBoot: Defines whether the instance has Secure Boot
	// enabled. Secure Boot helps ensure that the system only runs authentic
	// software by verifying the digital signature of all boot components,
	// and halting the boot process if signature verification fails.
	EnableSecureBoot bool `json:"enableSecureBoot,omitempty"`
}

// GKEUpgradeSettings These upgrade settings control the level of
// parallelism and the level of disruption caused by an upgrade.
// maxUnavailable controls the number of nodes that can be
// simultaneously unavailable. maxSurge controls the number of
// additional nodes that can be added to the node pool temporarily for
// the time of the upgrade to increase the number of available nodes.
// (maxUnavailable + maxSurge) determines the level of parallelism (how
// many nodes are being upgraded at the same time). Note: upgrades
// inevitably introduce some disruption since workloads need to be moved
// from old nodes to new, upgraded ones. Even if maxUnavailable=0, this
// holds true. (Disruption stays within the limits of
// PodDisruptionBudget, if it is configured.) Consider a hypothetical
// node pool with 5 nodes having maxSurge=2, maxUnavailable=1. This
// means the upgrade process upgrades 3 nodes simultaneously. It creates
// 2 additional (upgraded) nodes, then it brings down 3 old (not yet
// upgraded) nodes at the same time. This ensures that there are always
// at least 4 nodes available.
type GKEUpgradeSettings struct {
	// MaxSurge: The maximum number of nodes that can be created beyond the
	// current size of the node pool during the upgrade process.
	MaxSurge int64 `json:"maxSurge,omitempty"`

	// MaxUnavailable: The maximum number of nodes that can be
	// simultaneously unavailable during the upgrade process. A node is
	// considered available if its status is Ready.
	MaxUnavailable int64 `json:"maxUnavailable,omitempty"`
}

// VirtualMachineInstancePresetList represents a list of VirtualMachineInstancePreset.
// swagger:model VirtualMachineInstancePresetList
type VirtualMachineInstancePresetList []VirtualMachineInstancePreset

// Need to copy the following type to avoid a collision on Resources
// between kubevirtv1.ResourceRequirements and corev1.ResourceRequirements used in different part of the API
type DomainSpec struct {
	// Resources describes the Compute Resources required by this vmi.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
	// CPU allow specified the detailed CPU topology inside the vmi.
	// +optional
	CPU *kubevirtv1.CPU `json:"cpu,omitempty"`
	// Memory allow specifying the VMI memory features.
	// +optional
	Memory *kubevirtv1.Memory `json:"memory,omitempty"`
	// Machine type.
	// +optional
	Machine *kubevirtv1.Machine `json:"machine,omitempty"`
	// Firmware.
	// +optional
	Firmware *kubevirtv1.Firmware `json:"firmware,omitempty"`
	// Clock sets the clock and timers of the vmi.
	// +optional
	Clock *kubevirtv1.Clock `json:"clock,omitempty"`
	// Features like acpi, apic, hyperv, smm.
	// +optional
	Features *kubevirtv1.Features `json:"features,omitempty"`
	// Devices allows adding disks, network interfaces, and others
	Devices kubevirtv1.Devices `json:"devices"`
	// Controls whether or not disks will share IOThreads.
	// Omitting IOThreadsPolicy disables use of IOThreads.
	// One of: shared, auto
	// +optional
	IOThreadsPolicy *kubevirtv1.IOThreadsPolicy `json:"ioThreadsPolicy,omitempty"`
	// Chassis specifies the chassis info passed to the domain.
	// +optional
	Chassis *kubevirtv1.Chassis `json:"chassis,omitempty"`
}

type VirtualMachineInstancePresetSpec struct {
	// Selector is a label query over a set of VMIs.
	// Required.
	Selector metav1.LabelSelector `json:"selector"`
	// Domain is the same object type as contained in VirtualMachineInstanceSpec
	Domain *DomainSpec `json:"domain,omitempty"`
}

// VirtualMachineInstancePreset represents a KubeVirt Virtual Machine Instance Preset
// swagger:model VirtualMachineInstancePreset
type VirtualMachineInstancePreset struct {
	apiv1.ObjectMeta `json:",inline"`
	// VirtualMachineInstance Spec contains the VirtualMachineInstance specification.
	Spec VirtualMachineInstancePresetSpec `json:"spec,omitempty" valid:"required"`
}

// StorageClassList represents a list of Kubernetes StorageClass.
// swagger:model StorageClassList
type StorageClassList []StorageClass

// StorageClass represents a Kubernetes StorageClass
// swagger:model StorageClass
type StorageClass struct {
	apiv1.ObjectMeta `json:",inline"`
	// Provisioner indicates the type of the provisioner.
	Provisioner string `json:"provisioner"`

	// Parameters holds the parameters for the provisioner that should
	// create volumes of this storage class.
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`

	// Dynamically provisioned PersistentVolumes of this storage class are
	// created with this reclaimPolicy. Defaults to Delete.
	// +optional
	ReclaimPolicy *v1.PersistentVolumeReclaimPolicy `json:"reclaimPolicy,omitempty"`

	// Dynamically provisioned PersistentVolumes of this storage class are
	// created with these mountOptions, e.g. ["ro", "soft"]. Not validated -
	// mount of the PVs will simply fail if one is invalid.
	// +optional
	MountOptions []string `json:"mountOptions,omitempty"`

	// AllowVolumeExpansion shows whether the storage class allow volume expand
	// +optional
	AllowVolumeExpansion *bool `json:"allowVolumeExpansion,omitempty"`

	// VolumeBindingMode indicates how PersistentVolumeClaims should be
	// provisioned and bound.  When unset, VolumeBindingImmediate is used.
	// This field is only honored by servers that enable the VolumeScheduling feature.
	// +optional
	VolumeBindingMode *storagev1.VolumeBindingMode `json:"volumeBindingMode,omitempty"`

	// Restrict the node topologies where volumes can be dynamically provisioned.
	// Each volume plugin defines its own supported topology specifications.
	// An empty TopologySelectorTerm list means there is no topology restriction.
	// This field is only honored by servers that enable the VolumeScheduling feature.
	// +optional
	// +listType=atomic
	AllowedTopologies []v1.TopologySelectorTerm `json:"allowedTopologies,omitempty"`
}

// CNIVersions is a list of versions for a CNI Plugin
// swagger:model CNIVersions
type CNIVersions struct {
	// CNIPluginType represents the type of the CNI Plugin
	CNIPluginType string
	// Versions represents the list of the CNI Plugin versions that are supported
	Versions []string
}
