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
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
)

// ConstraintTemplate represents a gatekeeper ConstraintTemplate
// swagger:model ConstraintTemplate
type ConstraintTemplate struct {
	Name string `json:"name"`

	Spec   kubermaticv1.ConstraintTemplateSpec `json:"spec"`
	Status v1beta1.ConstraintTemplateStatus    `json:"status"`
}

// Constraint represents a gatekeeper Constraint
// swagger:model Constraint
type Constraint struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`

	Spec   kubermaticv1.ConstraintSpec `json:"spec"`
	Status *ConstraintStatus           `json:"status,omitempty"`
}

// ConstraintStatus represents a constraint status which holds audit info.
type ConstraintStatus struct {
	Enforcement    string      `json:"enforcement,omitempty"`
	AuditTimestamp string      `json:"auditTimestamp,omitempty"`
	Violations     []Violation `json:"violations,omitempty"`
	Synced         *bool       `json:"synced,omitempty"`
}

// Violation represents a gatekeeper constraint violation.
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

// GVK group version kind of a resource.
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
	Name    kubermaticv1.ProviderType `json:"name"`
	Enabled bool                      `json:"enabled"`
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
	Metering kubermaticv1.MeteringConfiguration `json:"metering"`
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
	apiv1.ObjectMeta

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

	Spec kubermaticv1.ClusterTemplateInstanceSpec `json:"spec"`
}

// RuleGroup represents a rule group of recording and alerting rules.
// swagger:model RuleGroup
type RuleGroup struct {
	Name string `json:"name"`
	// IsDefault indicates whether the ruleGroup is default
	IsDefault bool `json:"isDefault,omitempty"`
	// contains the RuleGroup data. Ref: https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/#rule_group
	Data []byte `json:"data"`
	// the type of this ruleGroup applies to. It can be `Metrics`.
	Type kubermaticv1.RuleGroupType `json:"type"`
}

// AllowedRegistry represents a object containing a allowed image registry prefix
// swagger:model AllowedRegistry
type AllowedRegistry struct {
	Name string `json:"name"`

	Spec kubermaticv1.AllowedRegistrySpec `json:"spec"`
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
	ScheduledTime      *apiv1.Time                    `json:"scheduledTime,omitempty"`
	BackupName         string                         `json:"backupName,omitempty"`
	JobName            string                         `json:"jobName,omitempty"`
	BackupStartTime    *apiv1.Time                    `json:"backupStartTime,omitempty"`
	BackupFinishedTime *apiv1.Time                    `json:"backupFinishedTime,omitempty"`
	BackupPhase        kubermaticv1.BackupStatusPhase `json:"backupPhase,omitempty"`
	BackupMessage      string                         `json:"backupMessage,omitempty"`
	DeleteJobName      string                         `json:"deleteJobName,omitempty"`
	DeleteStartTime    *apiv1.Time                    `json:"deleteStartTime,omitempty"`
	DeleteFinishedTime *apiv1.Time                    `json:"deleteFinishedTime,omitempty"`
	DeletePhase        kubermaticv1.BackupStatusPhase `json:"deletePhase,omitempty"`
	DeleteMessage      string                         `json:"deleteMessage,omitempty"`
}

type EtcdBackupConfigCondition struct {
	// Type of EtcdBackupConfig condition.
	Type kubermaticv1.EtcdBackupConfigConditionType `json:"type"`
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
	apiv1.ObjectMeta

	Name string `json:"name"`

	Spec   EtcdRestoreSpec   `json:"spec"`
	Status EtcdRestoreStatus `json:"status"`
}

type EtcdRestoreStatus struct {
	Phase       kubermaticv1.EtcdRestorePhase `json:"phase"`
	RestoreTime *apiv1.Time                   `json:"restoreTime,omitempty"`
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
	MonitoringRateLimits *kubermaticv1.MonitoringRateLimitSettings `json:"monitoringRateLimits,omitempty"`
	// LoggingRateLimits contains rate-limiting configuration logging in the user cluster.
	LoggingRateLimits *kubermaticv1.LoggingRateLimitSettings `json:"loggingRateLimits,omitempty"`
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

	// STOPPED state indicates the cluster is stopped, this state is specific to aks clusters.
	STOPPED ExternalClusterState = "STOPPED"

	// STOPPING state indicates the cluster is stopping, this state is specific to aks clusters.
	STOPPING ExternalClusterState = "STOPPING"

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

// ExternalClusterStatus defines the external cluster status.
type ExternalClusterStatus struct {
	State         ExternalClusterState `json:"state"`
	StatusMessage string               `json:"statusMessage,omitempty"`
}

// ExternalClusterSpec defines the external cluster specification.
type ExternalClusterSpec struct {
	// Version desired version of the kubernetes master components
	Version ksemver.Semver `json:"version,omitempty"`
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
	Name            string          `json:"name"`
	AccessKeyID     string          `json:"accessKeyID,omitempty" required:"true"`
	SecretAccessKey string          `json:"secretAccessKey,omitempty" required:"true"`
	Region          string          `json:"region" required:"true"`
	ClusterSpec     *EKSClusterSpec `json:"clusterSpec,omitempty"`
}

type EKSClusterSpec struct {
	// The VPC configuration used by the cluster control plane. Amazon EKS VPC resources
	// have specific requirements to work properly with Kubernetes. For more information,
	// see Cluster VPC Considerations (https://docs.aws.amazon.com/eks/latest/userguide/network_reqs.html)
	// and Cluster Security Group Considerations (https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html)
	// in the Amazon EKS User Guide. You must specify at least two subnets. You
	// can specify up to five security groups, but we recommend that you use a dedicated
	// security group for your cluster control plane.
	//
	// ResourcesVpcConfig is a required field

	ResourcesVpcConfig VpcConfigRequest `json:"vpcConfigRequest" required:"true"`

	// The desired Kubernetes version for your cluster. If you don't specify a value
	// here, the latest version available in Amazon EKS is used.
	Version string `json:"version"`

	// The Amazon Resource Name (ARN) of the IAM role that provides permissions
	// for the Kubernetes control plane to make calls to AWS API operations on your
	// behalf. For more information, see Amazon EKS Service IAM Role (https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html)
	// in the Amazon EKS User Guide .
	//
	// RoleArn is a required field
	RoleArn string `json:"roleArn" required:"true"`
}

type AKSCloudSpec struct {
	Name           string          `json:"name"`
	TenantID       string          `json:"tenantID,omitempty" required:"true"`
	SubscriptionID string          `json:"subscriptionID,omitempty" required:"true"`
	ClientID       string          `json:"clientID,omitempty" required:"true"`
	ClientSecret   string          `json:"clientSecret,omitempty" required:"true"`
	ResourceGroup  string          `json:"resourceGroup" required:"true"`
	ClusterSpec    *AKSClusterSpec `json:"clusterSpec,omitempty"`
}

// AKSClusterSpec Azure Kubernetes Service cluster.
type AKSClusterSpec struct {
	// Location - Resource location
	Location string `json:"location" required:"true"`
	// KubernetesVersion - When you upgrade a supported AKS cluster, Kubernetes minor versions cannot be skipped. All upgrades must be performed sequentially by major version number. For example, upgrades between 1.14.x -> 1.15.x or 1.15.x -> 1.16.x are allowed, however 1.14.x -> 1.16.x is not allowed. See [upgrading an AKS cluster](https://docs.microsoft.com/azure/aks/upgrade-cluster) for more details.
	KubernetesVersion string `json:"kubernetesVersion"`
	// NodeResourceGroup - The name of the resource group containing agent pool nodes.
	NodeResourceGroup string `json:"nodeResourceGroup,omitempty"`
	// EnableRBAC - Whether Kubernetes Role-Based Access Control Enabled.
	EnableRBAC bool `json:"enableRBAC,omitempty"`
	// ManagedAAD - Whether The Azure Active Directory configuration Enabled.
	ManagedAAD bool `json:"managedAAD,omitempty"`
	// DNSPrefix - This cannot be updated once the Managed Cluster has been created.
	DNSPrefix string `json:"dnsPrefix,omitempty"`
	// FqdnSubdomain - This cannot be updated once the Managed Cluster has been created.
	FqdnSubdomain string `json:"fqdnSubdomain,omitempty"`
	// Fqdn - READ-ONLY; The FQDN of the master pool.
	Fqdn string `json:"fqdn,omitempty"`
	// PrivateFQDN - READ-ONLY; The FQDN of private cluster.
	PrivateFQDN string `json:"privateFQDN,omitempty"`
	// MachineDeploymentSpec - The agent pool properties.
	MachineDeploymentSpec *AKSMachineDeploymentCloudSpec `json:"machineDeploymentSpec,omitempty"`
	// NetworkProfile - The network configuration profile.
	NetworkProfile AKSNetworkProfile `json:"networkProfile,omitempty"`
}

// AKS NetworkProfile profile of network configuration.
type AKSNetworkProfile struct {
	// PodCidr - A CIDR notation IP range from which to assign pod IPs when kubenet is used.
	PodCidr string `json:"podCidr,omitempty"`
	// ServiceCidr - A CIDR notation IP range from which to assign service cluster IPs. It must not overlap with any Subnet IP ranges.
	ServiceCidr string `json:"serviceCidr,omitempty"`
	// DNSServiceIP - An IP address assigned to the Kubernetes DNS service. It must be within the Kubernetes service address range specified in serviceCidr.
	DNSServiceIP string `json:"dnsServiceIP,omitempty"`
	// DockerBridgeCidr - A CIDR notation IP range assigned to the Docker bridge network. It must not overlap with any Subnet IP ranges or the Kubernetes service address range.
	DockerBridgeCidr string `json:"dockerBridgeCidr,omitempty"`
	// NetworkPlugin - Network plugin used for building the Kubernetes network. Possible values include: 'Azure', 'Kubenet'
	NetworkPlugin string `json:"networkPlugin,omitempty"`
	// NetworkPolicy - Network policy used for building the Kubernetes network. Possible values include: 'Calico', 'Azure'
	NetworkPolicy string `json:"networkPolicy,omitempty"`
	// NetworkMode - This cannot be specified if networkPlugin is anything other than 'azure'. Possible values include: 'Transparent', 'Bridge'
	NetworkMode string `json:"networkMode,omitempty"`
	// OutboundType - This can only be set at cluster creation time and cannot be changed later. For more information see [egress outbound type](https://docs.microsoft.com/azure/aks/egress-outboundtype). Possible values include: 'OutboundTypeLoadBalancer', 'OutboundTypeUserDefinedRouting', 'OutboundTypeManagedNATGateway', 'OutboundTypeUserAssignedNATGateway'
	OutboundType string `json:"outboundType,omitempty"`
	// LoadBalancerSku - The default is 'standard'. See [Azure Load Balancer SKUs](https://docs.microsoft.com/azure/load-balancer/skus) for more information about the differences between load balancer SKUs. Possible values include: 'LoadBalancerSkuStandard', 'LoadBalancerSkuBasic'
	LoadBalancerSku string `json:"loadBalancerSku,omitempty"`
}

type VpcConfigRequest struct {
	// Specify one or more security groups for the cross-account elastic network
	// interfaces that Amazon EKS creates to use to allow communication between
	// your nodes and the Kubernetes control plane. If you don't specify any security
	// groups, then familiarize yourself with the difference between Amazon EKS
	// defaults for clusters deployed with Kubernetes:
	//
	//    * 1.14 Amazon EKS platform version eks.2 and earlier
	//
	//    * 1.14 Amazon EKS platform version eks.3 and later
	//
	// For more information, see Amazon EKS security group considerations (https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html)
	// in the Amazon EKS User Guide .
	SecurityGroupIds []*string `json:"securityGroupIds" required:"true"`

	// Specify subnets for your Amazon EKS nodes. Amazon EKS creates cross-account
	// elastic network interfaces in these subnets to allow communication between
	// your nodes and the Kubernetes control plane.
	SubnetIds []*string `json:"subnetIds" required:"true"`
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

// Regions represents an list of EKS regions.
// swagger:model EKSRegions
type EKSRegions []string

// EKSSubnetIDList represents an array of EKS subnetID.
// swagger:model EKSSubnetIDList
type EKSSubnetIDList []EKSSubnetID

// EKSSubnetID represents a object of EKS subnetID.
// swagger:model EKSSubnetID
type EKSSubnetID string

// EKSSecurityGroupIDList represents an array of EKS securityGroupID.
// swagger:model EKSSecurityGroupIDList
type EKSSecurityGroupIDList []EKSSecurityGroupID

// EKSSecurityGroupID represents a object of EKS securityGroupID.
// swagger:model EKSSecurityGroupID
type EKSSecurityGroupID string

// EKSVPCList represents an array of EKS VPC.
// swagger:model EKSVPCList
type EKSVPCList []EKSVPC

// EKSVPC represents a object of EKS VpcId.
// swagger:model EKSVPC
type EKSVPC struct {
	ID        string `json:"id"`
	IsDefault bool   `json:"default"`
}

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
	// Name - Node pool name must contain only lowercase letters and numbers. For Linux node pools must be 12 or fewer characters.
	Name string `json:"name"`
	// BasicSettings - Settings for creating the AKS agentpool
	BasicSettings AgentPoolBasics `json:"basicSettings"`
	// OptionalSettings - Optional Settings for creating the AKS agentpool
	OptionalSettings AgentPoolOptionalSettings `json:"optionalSettings,omitempty"`
	// Configuration - Configuration of created AKS agentpool
	Configuration AgentPoolConfig `json:"configuration,omitempty"`
}

type AgentPoolBasics struct {
	// Required: Count - Number of agents (VMs) to host docker containers. Allowed values must be in the range of 0 to 1000 (inclusive) for user pools and in the range of 1 to 1000 (inclusive) for system pools. The default value is 1.
	Count int32 `json:"count" required:"true"`
	// Required: VMSize - VM size availability varies by region. If a node contains insufficient compute resources (memory, cpu, etc) pods might fail to run correctly. For more details on restricted VM sizes, see: https://docs.microsoft.com/azure/aks/quotas-skus-regions
	VMSize string `json:"vmSize" required:"true"`
	// Mode - Possible values include: 'System', 'User'.
	Mode string `json:"mode,omitempty"`
	// OrchestratorVersion - As a best practice, you should upgrade all node pools in an AKS cluster to the same Kubernetes version. The node pool version must have the same major version as the control plane. The node pool minor version must be within two minor versions of the control plane version. The node pool version cannot be greater than the control plane version. For more information see [upgrading a node pool](https://docs.microsoft.com/azure/aks/use-multiple-node-pools#upgrade-a-node-pool).
	OrchestratorVersion string `json:"orchestratorVersion,omitempty"`
	// AvailabilityZones - The list of Availability zones to use for nodes. This can only be specified if the AgentPoolType property is 'VirtualMachineScaleSets'.
	AvailabilityZones []string `json:"availabilityZones,omitempty"`
	// EnableAutoScaling - Whether to enable auto-scaler
	EnableAutoScaling bool `json:"enableAutoScaling,omitempty"`
	// The scaling configuration details for the Auto Scaling group that is created
	// for your node group.
	ScalingConfig NodegroupScalingConfig `json:"scalingConfig,omitempty"`
	// The OSDiskSize for Agent agentpool cannot be less than 30GB or larger than 2048GB.
	OsDiskSizeGB int32 `json:"osDiskSizeGB,omitempty"`
}

type NodegroupScalingConfig struct {
	// MaxCount - The maximum number of nodes for auto-scaling
	MaxCount int32 `json:"maxCount,omitempty"`
	// MinCount - The minimum number of nodes for auto-scaling
	MinCount int32 `json:"minCount,omitempty"`
}

type AgentPoolOptionalSettings struct {
	// NodeLabels - The node labels to be persisted across all nodes in agent pool.
	NodeLabels map[string]*string `json:"nodeLabels,omitempty"`
	// NodeTaints - The taints added to new nodes during node pool create and scale. For example, key=value:NoSchedule.
	// Placing custom taints on system pool is not supported(except 'CriticalAddonsOnly' taint or taint effect is 'PreferNoSchedule'). Please refer to https://aka.ms/aks/system-taints for detail
	NodeTaints []string `json:"nodeTaints,omitempty"`
}

type AgentPoolConfig struct {
	// OsDiskType - Possible values include: 'Managed', 'Ephemeral'
	OsDiskType string `json:"osDiskType,omitempty"`
	// MaxPods - The maximum number of pods that can run on a node.
	MaxPods int32 `json:"maxPods,omitempty"`
	// OsType - Possible values include: 'Linux', 'Windows'. The default value is 'Linux'.
	// Windows node pools are not supported on kubenet clusters
	OsType string `json:"osType,omitempty"`
	// EnableNodePublicIP - Some scenarios may require nodes in a node pool to receive their own dedicated public IP addresses. A common scenario is for gaming workloads, where a console needs to make a direct connection to a cloud virtual machine to minimize hops. For more information see [assigning a public IP per node](https://docs.microsoft.com/azure/aks/use-multiple-node-pools#assign-a-public-ip-per-node-for-your-node-pools). The default is false.
	EnableNodePublicIP bool `json:"enableNodePublicIP,omitempty"`
	// MaxSurgeUpgradeSetting - This can either be set to an integer (e.g. '5') or a percentage (e.g. '50%'). If a percentage is specified, it is the percentage of the total agent pool size at the time of the upgrade. For percentages, fractional nodes are rounded up. If not specified, the default is 1. For more information, including best practices, see: https://docs.microsoft.com/azure/aks/upgrade-cluster#customize-node-surge-upgrade
	MaxSurgeUpgradeSetting string `json:"maxSurge,omitempty"`
	// VnetSubnetID - If this is not specified, a VNET and subnet will be generated and used. If no podSubnetID is specified, this applies to nodes and pods, otherwise it applies to just nodes. This is of the form: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets/{subnetName}
	VnetSubnetID string `json:"vnetSubnetID,omitempty"`
	// PodSubnetID - If omitted, pod IPs are statically assigned on the node subnet (see vnetSubnetID for more details). This is of the form: /subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/virtualNetworks/{virtualNetworkName}/subnets/{subnetName}
	PodSubnetID string `json:"podSubnetID,omitempty"`
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

// VirtualMachineInstancePreset represents a KubeVirt Virtual Machine Instance Preset
// swagger:model VirtualMachineInstancePreset
type VirtualMachineInstancePreset struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	// Spec contains the kubevirtv1.VirtualMachineInstancePreset.Spec object marshalled
	Spec string `json:"spec,omitempty"`
}

// StorageClassList represents a list of Kubernetes StorageClass.
// swagger:model StorageClassList
type StorageClassList []StorageClass

// StorageClass represents a Kubernetes StorageClass
// swagger:model StorageClass
type StorageClass struct {
	Name string `json:"name"`
}

// CNIVersions is a list of versions for a CNI Plugin
// swagger:model CNIVersions
type CNIVersions struct {
	// CNIPluginType represents the type of the CNI Plugin
	CNIPluginType string `json:"cniPluginType"`
	// Versions represents the list of the CNI Plugin versions that are supported
	Versions []string `json:"versions"`
}
