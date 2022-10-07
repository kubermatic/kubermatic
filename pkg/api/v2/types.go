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
	"time"

	constrainttemplatesv1 "github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ConstraintTemplate represents a gatekeeper ConstraintTemplate
// swagger:model ConstraintTemplate
type ConstraintTemplate struct {
	Name string `json:"name"`

	Spec   kubermaticv1.ConstraintTemplateSpec            `json:"spec"`
	Status constrainttemplatesv1.ConstraintTemplateStatus `json:"status"`
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

// PresetBody represents the body of a created preset
// swagger:model PresetBody
type PresetBody struct {
	PresetBodyMetadata `json:"metadata,omitempty"`
	Spec               kubermaticv1.PresetSpec `json:"spec"`
}

// PresetBodyMetadata represents metadata within the body of a created preset
// swagger:model PresetBodyMetadata
type PresetBodyMetadata struct {
	Name string `json:"name,omitempty"`
}

// PresetProvider represents a preset provider
// swagger:model PresetProvider
type PresetProvider struct {
	Name    kubermaticv1.ProviderType `json:"name"`
	Enabled bool                      `json:"enabled"`
}

// PresetStats represents a preset statistics
// swagger:model PresetStats
type PresetStats struct {
	AssociatedClusters         int `json:"associatedClusters"`
	AssociatedClusterTemplates int `json:"associatedClusterTemplates"`
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
	ID   string `json:"id,omitempty"`

	ProjectID      string                         `json:"projectID,omitempty"`
	User           string                         `json:"user,omitempty"`
	Scope          string                         `json:"scope"`
	UserSSHKeys    []ClusterTemplateSSHKey        `json:"userSshKeys,omitempty"`
	Cluster        *ClusterTemplateInfo           `json:"cluster,omitempty"`
	NodeDeployment *ClusterTemplateNodeDeployment `json:"nodeDeployment,omitempty"`
	Applications   []apiv1.Application            `json:"applications,omitempty"`
}

// ClusterTemplateInfo represents a ClusterTemplateInfo object.
type ClusterTemplateInfo struct {
	Labels          map[string]string `json:"labels,omitempty"`
	InheritedLabels map[string]string `json:"inheritedLabels,omitempty"`
	// indicates the preset name
	Credential string            `json:"credential,omitempty"`
	Spec       apiv1.ClusterSpec `json:"spec"`
}

type ClusterTemplateNodeDeployment struct {
	Spec apiv1.NodeDeploymentSpec `json:"spec"`
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
	// swagger:strfmt date-time
	ScheduledTime *apiv1.Time `json:"scheduledTime,omitempty"`
	BackupName    string      `json:"backupName,omitempty"`
	JobName       string      `json:"jobName,omitempty"`
	// swagger:strfmt date-time
	BackupStartTime *apiv1.Time `json:"backupStartTime,omitempty"`
	// swagger:strfmt date-time
	BackupFinishedTime *apiv1.Time                    `json:"backupFinishedTime,omitempty"`
	BackupPhase        kubermaticv1.BackupStatusPhase `json:"backupPhase,omitempty"`
	BackupMessage      string                         `json:"backupMessage,omitempty"`
	DeleteJobName      string                         `json:"deleteJobName,omitempty"`
	// swagger:strfmt date-time
	DeleteStartTime *apiv1.Time `json:"deleteStartTime,omitempty"`
	// swagger:strfmt date-time
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
	// swagger:strfmt date-time
	LastHeartbeatTime apiv1.Time `json:"lastHeartbeatTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	// swagger:strfmt date-time
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
	// the cluster's Seed.Spec.EtcdBackupRestore.
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
	Phase kubermaticv1.EtcdRestorePhase `json:"phase"`
	// swagger:strfmt date-time
	RestoreTime *apiv1.Time `json:"restoreTime,omitempty"`
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
	// the cluster's Seed.Spec.EtcdBackupRestore.
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
	Spec             ExternalClusterSpec       `json:"spec,omitempty"`
	Cloud            *ExternalClusterCloudSpec `json:"cloud,omitempty"`
	Status           ExternalClusterStatus     `json:"status"`
}

type ExternalClusterState string
type ExternalClusterMDState string

const (
	// ProvisioningExternalClusterState state indicates the cluster is being created.
	ProvisioningExternalClusterState ExternalClusterState = "Provisioning"

	// StoppedExternalClusterState state indicates the cluster is stopped, this state is specific to AKS clusters.
	StoppedExternalClusterState ExternalClusterState = "Stopped"

	// StoppingExternalClusterState state indicates the cluster is stopping, this state is specific to AKS clusters.
	StoppingExternalClusterState ExternalClusterState = "Stopping"

	// RunningExternalClusterState state indicates the cluster has been created and is fully usable.
	RunningExternalClusterState ExternalClusterState = "Running"

	// ReconcilingExternalClusterState state indicates that some work is actively being done on the cluster, such as upgrading the master or
	// node software. Details can be found in the `StatusMessage` field.
	ReconcilingExternalClusterState ExternalClusterState = "Reconciling"

	// DeletingExternalClusterState state indicates the cluster is being deleted.
	DeletingExternalClusterState ExternalClusterState = "Deleting"

	// StartingExternalClusterState state indicates the cluster is starting.
	StartingExternalClusterState ExternalClusterState = "Starting"

	// UnknownExternalClusterState indicates undefined state.
	UnknownExternalClusterState ExternalClusterState = "Unknown"

	// ErrorExternalClusterState state indicates the cluster is unusable. It will be automatically deleted. Details can be found in the
	// `statusMessage` field.
	ErrorExternalClusterState ExternalClusterState = "Error"

	// WarningExternalClusterState state indicates the cluster is usable but with some warnings. Details can be found in the
	// `statusMessage` field.
	WarningExternalClusterState ExternalClusterState = "Warning"
)

const (
	// ProvisioningExternalClusterMDState state indicates the cluster machine deployment is being created.
	ProvisioningExternalClusterMDState ExternalClusterMDState = "Provisioning"

	// RunningExternalClusterMDState state indicates the cluster machine deployment has been created and is fully usable.
	RunningExternalClusterMDState ExternalClusterMDState = "Running"

	// StoppedExternalClusterMDState state indicates the cluster machine deployment is stopped. This state is specific to AKS clusters.
	StoppedExternalClusterMDState ExternalClusterMDState = "Stopped"

	// ReconcilingExternalClusterMDState state indicates that some work is actively being done on the machine deployment, such as upgrading the master or
	// node software. Details can be found in the `StatusMessage` field.
	ReconcilingExternalClusterMDState ExternalClusterMDState = "Reconciling"

	// DeletingExternalClusterMDState state indicates the machine deployment is being deleted.
	DeletingExternalClusterMDState ExternalClusterMDState = "Deleting"

	// StartingExternalClusterMDState state indicates the cluster machine deployment is starting.
	StartingExternalClusterMDState ExternalClusterMDState = "Starting"

	// UnknownExternalClusterMDState indicates undefined state.
	UnknownExternalClusterMDState ExternalClusterMDState = "Unknown"

	// ErrorExternalClusterMDState state indicates the machine deployment is unusable. It will be automatically deleted. Details can be found in the
	// `statusMessage` field.
	ErrorExternalClusterMDState ExternalClusterMDState = "Error"

	// WarningExternalClusterMDState state indicates the machine deployment is usable but with some warnings. Details can be found in the
	// `statusMessage` field.
	WarningExternalClusterMDState ExternalClusterMDState = "Warning"
)

// ExternalClusterStatus defines the external cluster status.
type ExternalClusterStatus struct {
	State         ExternalClusterState `json:"state"`
	StatusMessage string               `json:"statusMessage,omitempty"`
	AKS           *AKSClusterStatus    `json:"aks,omitempty"`
}

// ExternalClusterSpec defines the external cluster specification.
type ExternalClusterSpec struct {
	// Version desired version of the kubernetes master components
	Version ksemver.Semver `json:"version,omitempty"`

	GKEClusterSpec *GKEClusterSpec `json:"gkeclusterSpec,omitempty"`
	EKSClusterSpec *EKSClusterSpec `json:"eksclusterSpec,omitempty"`
	AKSClusterSpec *AKSClusterSpec `json:"aksclusterSpec,omitempty"`
}

// ExternalClusterCloudSpec represents an object holding cluster cloud details
// swagger:model ExternalClusterCloudSpec
type ExternalClusterCloudSpec struct {
	GKE          *GKECloudSpec     `json:"gke,omitempty"`
	EKS          *EKSCloudSpec     `json:"eks,omitempty"`
	AKS          *AKSCloudSpec     `json:"aks,omitempty"`
	KubeOne      *KubeOneSpec      `json:"kubeOne,omitempty"`
	BringYourOwn *BringYourOwnSpec `json:"bringYourOwn,omitempty"`
}

type BringYourOwnSpec struct{}

type KubeOneSpec struct {
	// Manifest Base64 encoded manifest
	Manifest         string            `json:"manifest,omitempty"`
	SSHKey           KubeOneSSHKey     `json:"sshKey,omitempty"`
	ContainerRuntime string            `json:"containerRuntime,omitempty"`
	CloudSpec        *KubeOneCloudSpec `json:"cloudSpec,omitempty"`
}

// SSHKeySpec represents the details of a ssh key.
type KubeOneSSHKey struct {
	// PrivateKey Base64 encoded privateKey
	PrivateKey string `json:"privateKey,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

type KubeOneCloudSpec struct {
	AWS                 *KubeOneAWSCloudSpec                 `json:"aws,omitempty"`
	GCP                 *KubeOneGCPCloudSpec                 `json:"gcp,omitempty"`
	Azure               *KubeOneAzureCloudSpec               `json:"azure,omitempty"`
	DigitalOcean        *KubeOneDigitalOceanCloudSpec        `json:"digitalocean,omitempty"`
	OpenStack           *KubeOneOpenStackCloudSpec           `json:"openstack,omitempty"`
	Equinix             *KubeOneEquinixCloudSpec             `json:"equinix,omitempty"`
	Hetzner             *KubeOneHetznerCloudSpec             `json:"hetzner,omitempty"`
	VSphere             *KubeOneVSphereCloudSpec             `json:"vsphere,omitempty"`
	VMwareCloudDirector *KubeOneVMwareCloudDirectorCloudSpec `json:"vmwareclouddirector,omitempty"`
	Nutanix             *KubeOneNutanixCloudSpec             `json:"nutanix,omitempty"`
}

// KubeOneAWSCloudSpec specifies access data to Amazon Web Services.
type KubeOneAWSCloudSpec struct {
	AccessKeyID     string `json:"accessKeyID"`
	SecretAccessKey string `json:"secretAccessKey"`
}

// KubeOneGCPCloudSpec specifies access data to GCP.
type KubeOneGCPCloudSpec struct {
	ServiceAccount string `json:"serviceAccount"`
}

// KubeOneAzureCloudSpec specifies access credentials to Azure cloud.
type KubeOneAzureCloudSpec struct {
	TenantID       string `json:"tenantID"`
	SubscriptionID string `json:"subscriptionID"`
	ClientID       string `json:"clientID"`
	ClientSecret   string `json:"clientSecret"`
}

// KubeOneDigitalOceanCloudSpec specifies access data to DigitalOcean.
type KubeOneDigitalOceanCloudSpec struct {
	// Token is used to authenticate with the DigitalOcean API.
	Token string `json:"token"`
}

// KubeOneOpenStackCloudSpec specifies access data to an OpenStack cloud.
type KubeOneOpenStackCloudSpec struct {
	AuthURL  string `json:"authURL"`
	Username string `json:"username"`
	Password string `json:"password"`

	// Project, formally known as tenant.
	Project string `json:"project"`
	// ProjectID, formally known as tenantID.
	ProjectID string `json:"projectID"`

	Domain string `json:"domain"`
	Region string `json:"region"`
}

// KubeOneVSphereCloudSpec credentials represents a credential for accessing vSphere.
type KubeOneVSphereCloudSpec struct {
	Server   string `json:"server"`
	Username string `json:"username"`
	Password string `json:"password"`
}

// KubeOneVMwareCloudDirectorCloudSpec represents credentials for accessing VMWare Cloud Director.
type KubeOneVMwareCloudDirectorCloudSpec struct {
	URL          string `json:"url"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Organization string `json:"organization"`
	VDC          string `json:"vdc"`
}

// KubeOneEquinixCloudSpec specifies access data to a Equinix cloud.
type KubeOneEquinixCloudSpec struct {
	APIKey    string `json:"apiKey"`
	ProjectID string `json:"projectID"`
}

// KubeOneHetznerCloudSpec specifies access data to hetzner cloud.
type KubeOneHetznerCloudSpec struct {
	// Token is used to authenticate with the Hetzner cloud API.
	Token string `json:"token"`
}

// KubeOneNutanixCloudSpec specifies the access data to Nutanix.
type KubeOneNutanixCloudSpec struct {
	Username string `json:"username"`
	Password string `json:"password"`
	// Endpoint is the Nutanix API (Prism Central) endpoint
	Endpoint string `json:"endpoint"`
	// Port is the Nutanix API (Prism Central) port
	Port string `json:"port"`

	// PrismElementUsername to be used for the CSI driver
	PrismElementUsername string `json:"elementUsername"`
	// PrismElementPassword to be used for the CSI driver
	PrismElementPassword string `json:"elementPassword"`
	// PrismElementEndpoint to access Nutanix Prism Element for the CSI driver
	PrismElementEndpoint string `json:"elementEndpoint"`

	// ClusterName is the Nutanix cluster that this user cluster will be deployed to.
	// +optional
	ClusterName   string `json:"clusterName,omitempty"`
	AllowInsecure bool   `json:"allowInsecure,omitempty"`
	ProxyURL      string `json:"proxyURL,omitempty"`
}

type GKECloudSpec struct {
	Name           string `json:"name"`
	ServiceAccount string `json:"serviceAccount,omitempty"`
	Zone           string `json:"zone"`
}

type EKSCloudSpec struct {
	Name            string `json:"name"`
	AccessKeyID     string `json:"accessKeyID,omitempty" required:"true"`
	SecretAccessKey string `json:"secretAccessKey,omitempty" required:"true"`
	Region          string `json:"region" required:"true"`
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

	// The Kubernetes network configuration for the cluster.
	KubernetesNetworkConfig *EKSKubernetesNetworkConfigResponse `json:"kubernetesNetworkConfig,omitempty"`

	// The desired Kubernetes version for your cluster. If you don't specify a value
	// here, the latest version available in Amazon EKS is used.
	Version string `json:"version,omitempty"`

	// The Unix epoch timestamp in seconds for when the cluster was created.
	CreatedAt *time.Time `json:"createdAt,omitempty"`

	// The metadata that you apply to the cluster to assist with categorization
	// and organization. Each tag consists of a key and an optional value. You define
	// both. Cluster tags do not propagate to any other resources associated with
	// the cluster.
	Tags map[string]*string `json:"tags,omitempty"`

	// The Amazon Resource Name (ARN) of the IAM role that provides permissions
	// for the Kubernetes control plane to make calls to AWS API operations on your
	// behalf. For more information, see Amazon EKS Service IAM Role (https://docs.aws.amazon.com/eks/latest/userguide/service_IAM_role.html)
	// in the Amazon EKS User Guide .
	//
	// RoleArn is a required field
	RoleArn string `json:"roleArn,omitempty" required:"true"`
}

// The Kubernetes network configuration for the cluster. The response contains
// a value for serviceIpv6Cidr or serviceIpv4Cidr, but not both.
type EKSKubernetesNetworkConfigResponse struct {
	// The IP family used to assign Kubernetes pod and service IP addresses. The
	// IP family is always ipv4, unless you have a 1.21 or later cluster running
	// version 1.10.1 or later of the Amazon VPC CNI add-on and specified ipv6 when
	// you created the cluster.
	IpFamily string `json:"ipFamily,omitempty"`

	// The CIDR block that Kubernetes pod and service IP addresses are assigned
	// from. Kubernetes assigns addresses from an IPv4 CIDR block assigned to a
	// subnet that the node is in. If you didn't specify a CIDR block when you created
	// the cluster, then Kubernetes assigns addresses from either the 10.100.0.0/16
	// or 172.20.0.0/16 CIDR blocks. If this was specified, then it was specified
	// when the cluster was created and it can't be changed.
	ServiceIpv4Cidr *string `json:"serviceIpv4Cidr,omitempty"`

	// The CIDR block that Kubernetes pod and service IP addresses are assigned
	// from if you created a 1.21 or later cluster with version 1.10.1 or later
	// of the Amazon VPC CNI add-on and specified ipv6 for ipFamily when you created
	// the cluster. Kubernetes assigns service addresses from the unique local address
	// range (fc00::/7) because you can't specify a custom IPv6 CIDR block when
	// you create the cluster.
	ServiceIpv6Cidr *string `json:"serviceIpv6Cidr,omitempty"`
}

type AKSCloudSpec struct {
	Name           string `json:"name"`
	TenantID       string `json:"tenantID,omitempty" required:"true"`
	SubscriptionID string `json:"subscriptionID,omitempty" required:"true"`
	ClientID       string `json:"clientID,omitempty" required:"true"`
	ClientSecret   string `json:"clientSecret,omitempty" required:"true"`
	ResourceGroup  string `json:"resourceGroup" required:"true"`
	Location       string `json:"location"`
}

// AKSClusterSpec Azure Kubernetes Service cluster.
type AKSClusterSpec struct {
	// The timestamp of resource creation (UTC).
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	// The identity that created the resource.
	CreatedBy *string `json:"createdBy,omitempty"`
	// KubernetesVersion - When you upgrade a supported AKS cluster, Kubernetes minor versions cannot be skipped. All upgrades must be performed sequentially by major version number. For example, upgrades between 1.14.x -> 1.15.x or 1.15.x -> 1.16.x are allowed, however 1.14.x -> 1.16.x is not allowed. See [upgrading an AKS cluster](https://docs.microsoft.com/azure/aks/upgrade-cluster) for more details.
	KubernetesVersion string `json:"kubernetesVersion"`
	// EnableRBAC - Whether Kubernetes Role-Based Access Control Enabled.
	EnableRBAC bool `json:"enableRBAC,omitempty"`
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
	// Resource tags.
	Tags map[string]*string `json:"tags,omitempty"`
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

type (
	AKSProvisioningState string
	AKSPowerState        string
)

type AKSClusterStatus struct {
	// ProvisioningState - Defines values for AKS cluster provisioning state.
	ProvisioningState AKSProvisioningState `json:"provisioningState"`
	// PowerState - Defines values for AKS cluster power state.
	PowerState AKSPowerState `json:"powerState"`
}

type VpcConfigRequest struct {
	// The VPC associated with your cluster.
	VpcId *string `json:"vpcId,omitempty"`

	// Specify one or more security groups for the cross-account elastic network
	// interfaces that Amazon EKS creates to use to allow communication between
	// your nodes and the Kubernetes control plane.
	// For more information, see Amazon EKS security group considerations (https://docs.aws.amazon.com/eks/latest/userguide/sec-group-reqs.html)
	// in the Amazon EKS User Guide .
	SecurityGroupIds []string `json:"securityGroupIds" required:"true"`

	// Specify subnets for your Amazon EKS nodes. Amazon EKS creates cross-account
	// elastic network interfaces in these subnets to allow communication between
	// your nodes and the Kubernetes control plane.
	SubnetIds []string `json:"subnetIds" required:"true"`
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
	Phase                ExternalClusterMDPhase                     `json:"phase"`
}

// ExternalClusterMDPhase defines the external cluster machinedeployment phase.
type ExternalClusterMDPhase struct {
	State         ExternalClusterMDState `json:"state"`
	StatusMessage string                 `json:"statusMessage,omitempty"`
	AKS           *AKSMDPhase            `json:"aks,omitempty"`
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
	IsDefault bool   `json:"default,omitempty"`
}

// GKEImageList represents an array of GKE images.
// swagger:model GKEImageList
type GKEImageList []GKEImage

// GKEDiskTypeList represents an array of GKE disk types.
// swagger:model GKEDiskTypeList
type GKEDiskTypeList []GKEDiskType

// GKEDiskType represents a object of GKE disk type.
// swagger:model GKEDiskType
type GKEDiskType struct {
	// Name of the resource.
	Name string `json:"name"`
	// Description: An optional description of this resource.
	Description string `json:"description,omitempty"`
	// DefaultDiskSizeGb: Server-defined default disk size in GB.
	DefaultDiskSizeGb int64 `json:"defaultDiskSizeGb,omitempty"`
	// Kind: Type of the resource. Always compute#diskType for
	// disk types.
	Kind string `json:"kind,omitempty"`
}

// GKEZone represents a object of GKE zone.
// swagger:model GKEZone
type GKEZone struct {
	Name      string `json:"name"`
	IsDefault bool   `json:"default,omitempty"`
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

// EKSClusterList represents a list of EKS clusters.
// swagger:model EKSClusterList
type EKSClusterList []EKSCluster

// EKSRegionList represents a list of EKS regions.
// swagger:model EKSRegionList
type EKSRegionList []string

// EKSClusterRole represents a EKS Cluster Service Role.
// swagger:model EKSClusterRole
type EKSClusterRole struct {
	// RoleName  represents the friendly name that identifies the role.
	RoleName string `json:"roleName"`

	// The Amazon Resource Name (ARN) specifying the role. For more information
	// about ARNs and how to use them in policies, see IAM identifiers (https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
	// in the IAM User Guide guide.
	Arn string `json:"arn"`
}

// EKSClusterRoleList represents a list of EKS Cluster Service Roles.
// swagger:model EKSClusterRoleList
type EKSClusterRoleList []EKSClusterRole

// EKSNodeRole represents a EKS Node IAM Role.
// swagger:model EKSNodeRole
type EKSNodeRole struct {
	// RoleName  represents the friendly name that identifies the role.
	RoleName string `json:"roleName"`

	// The Amazon Resource Name (ARN) specifying the role. For more information
	// about ARNs and how to use them in policies, see IAM identifiers (https://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html)
	// in the IAM User Guide guide.
	Arn string `json:"arn"`
}

// EKSNodeRoleList represents a list of EKS Node IAM Roles.
// swagger:model EKSNodeRoleList
type EKSNodeRoleList []EKSNodeRole

// EKSAMITypeList represents a list of EKS AMI Types for node group.
// swagger:model EKSAMITypeList
type EKSAMITypeList []string

// EKSCapacityTypeList represents a list of EKS Capacity Types for node group.
// swagger:model EKSCapacityTypeList
type EKSCapacityTypeList []string

// EKSInstanceTypeList represents a list of EKS InstanceType object for node group.
// swagger:model EKSInstanceTypeList
type EKSInstanceTypeList []EKSInstanceType

// EKSInstanceType is the object representing EKS nodegroup instancetype..
// swagger:model EKSInstanceType
type EKSInstanceType struct {
	Name         string  `json:"name"`
	PrettyName   string  `json:"pretty_name,omitempty"`
	Memory       float32 `json:"memory,omitempty"`
	VCPUs        int     `json:"vcpus,omitempty"`
	GPUs         int     `json:"gpus,omitempty"`
	Architecture string  `json:"architecture,omitempty"`
}

// EKSSubnetList represents an array of EKS subnet.
// swagger:model EKSSubnetList
type EKSSubnetList []EKSSubnet

// EKSSubnet represents a object of EKS subnet.
// swagger:model EKSSubnet
type EKSSubnet struct {
	// The Availability Zone of the subnet.
	AvailabilityZone string `json:"availabilityZone"`
	// The ID of the subnet.
	SubnetId string `json:"subnetId"`
	// The ID of the VPC the subnet is in.
	VpcId   string `json:"vpcId"`
	Default bool   `json:"default"`
}

// EKSSecurityGroupList represents an array of EKS securityGroup.
// swagger:model EKSSecurityGroupList
type EKSSecurityGroupList []EKSSecurityGroup

// EKSSecurityGroup represents a object of EKS securityGroup.
// swagger:model EKSSecurityGroup
type EKSSecurityGroup struct {
	// The ID of the security group.
	GroupId string `json:"groupId"`
	// [VPC only] The ID of the VPC for the security group.
	VpcId string `json:"vpcId"`
}

// EKSVPCList represents an array of EKS VPC.
// swagger:model EKSVPCList
type EKSVPCList []EKSVPC

// EKSVPC represents a object of EKS VpcId.
// swagger:model EKSVPC
type EKSVPC struct {
	ID        string `json:"id"`
	IsDefault bool   `json:"default"`
}

// AKSCluster represents an object of AKS cluster.
// swagger:model AKSCluster
type AKSCluster struct {
	Name          string `json:"name"`
	ResourceGroup string `json:"resourceGroup"`
	Location      string `json:"location"`
	IsImported    bool   `json:"imported"`
}

// AKSClusterList represents an list of AKS clusters.
// swagger:model AKSClusterList
type AKSClusterList []AKSCluster

// AKSVMSizeList represents an array of AKS VM sizes.
// swagger:model AKSVMSizeList
type AKSVMSizeList []AKSVMSize

// AKSLocationList represents a list of AKS Locations.
// swagger:model AKSLocationList
type AKSLocationList []AKSLocation

// AKSLocation represents an object of Azure Location.
// swagger:model AKSLocation
type AKSLocation struct {
	// The location name.
	Name string `json:"name,omitempty"`
	// READ-ONLY; The category of the region.
	RegionCategory string `json:"regionCategory,omitempty"`
}

// AzureResourceGroup represents an object of Azure ResourceGroup information.
type AzureResourceGroup struct {
	// The name of the resource group.
	Name string `json:"name,omitempty"`
}

// AzureResourceGroupList represents an list of AKS ResourceGroups.
// swagger:model AzureResourceGroupList
type AzureResourceGroupList []AzureResourceGroup

// AKSVMSize is the object representing Azure VM sizes.
// swagger:model AKSVMSize
type AKSVMSize struct {
	Name                 string `json:"name,omitempty"`
	NumberOfCores        int32  `json:"numberOfCores,omitempty"`
	NumberOfGPUs         int32  `json:"numberOfGPUs,omitempty"`
	OsDiskSizeInMB       int32  `json:"osDiskSizeInMB,omitempty"`
	ResourceDiskSizeInMB int32  `json:"resourceDiskSizeInMB,omitempty"`
	MemoryInMB           int32  `json:"memoryInMB,omitempty"`
	MaxDataDiskCount     int32  `json:"maxDataDiskCount,omitempty"`
}

// AKSNodePoolModes represents nodepool modes.
// swagger:model AKSNodePoolModes
type AKSNodePoolModes []string

// FeatureGates represents an object holding feature gate settings
// swagger:model FeatureGates
type FeatureGates struct {
	KonnectivityService    *bool `json:"konnectivityService,omitempty"`
	OIDCKubeCfgEndpoint    *bool `json:"oidcKubeCfgEndpoint,omitempty"`
	OperatingSystemManager *bool `json:"operatingSystemManager,omitempty"`
}

// ExternalClusterMachineDeploymentCloudSpec represents an object holding machine deployment cloud details.
// swagger:model ExternalClusterMachineDeploymentCloudSpec
type ExternalClusterMachineDeploymentCloudSpec struct {
	GKE *GKEMachineDeploymentCloudSpec `json:"gke,omitempty"`
	AKS *AKSMachineDeploymentCloudSpec `json:"aks,omitempty"`
	EKS *EKSMachineDeploymentCloudSpec `json:"eks,omitempty"`
}

type EKSMachineDeploymentCloudSpec struct {
	// The Unix epoch timestamp in seconds for when the managed node group was created.
	CreatedAt time.Time `json:"createdAt,omitempty"`

	// The subnets to use for the Auto Scaling group that is created for your node
	// group. These subnets must have the tag key kubernetes.io/cluster/CLUSTER_NAME
	// with a value of shared, where CLUSTER_NAME is replaced with the name of your
	// cluster. If you specify launchTemplate, then don't specify SubnetId (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateNetworkInterface.html)
	// in your launch template, or the node group deployment will fail. For more
	// information about using launch templates with Amazon EKS, see Launch template
	// support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	//
	// Subnets is a required field
	Subnets []string `json:"subnets" required:"true"`

	// The Amazon Resource Name (ARN) of the IAM role to associate with your node
	// group. The Amazon EKS worker node kubelet daemon makes calls to AWS APIs
	// on your behalf. Nodes receive permissions for these API calls through an
	// IAM instance profile and associated policies. Before you can launch nodes
	// and register them into a cluster, you must create an IAM role for those nodes
	// to use when they are launched. For more information, see Amazon EKS node
	// IAM role (https://docs.aws.amazon.com/eks/latest/userguide/worker_node_IAM_role.html)
	// in the Amazon EKS User Guide . If you specify launchTemplate, then don't
	// specify IamInstanceProfile (https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_IamInstanceProfile.html)
	// in your launch template, or the node group deployment will fail. For more
	// information about using launch templates with Amazon EKS, see Launch template
	// support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	//
	// NodeRole is a required field
	NodeRole string `json:"nodeRole" required:"true"`

	// The AMI type for your node group. GPU instance types should use the AL2_x86_64_GPU
	// AMI type. Non-GPU instances should use the AL2_x86_64 AMI type. Arm instances
	// should use the AL2_ARM_64 AMI type. All types use the Amazon EKS optimized
	// Amazon Linux 2 AMI. If you specify launchTemplate, and your launch template
	// uses a custom AMI, then don't specify amiType, or the node group deployment
	// will fail. For more information about using launch templates with Amazon
	// EKS, see Launch template support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	AmiType string `json:"amiType,omitempty"`

	// The architecture of the machine image.
	Architecture string `json:"architecture,omitempty"`

	// The capacity type for your node group. Possible values ON_DEMAND | SPOT
	CapacityType string `json:"capacityType,omitempty"`

	// The root device disk size (in GiB) for your node group instances. The default
	// disk size is 20 GiB. If you specify launchTemplate, then don't specify diskSize,
	// or the node group deployment will fail. For more information about using
	// launch templates with Amazon EKS, see Launch template support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	DiskSize int32 `json:"diskSize,omitempty"`

	// Specify the instance types for a node group. If you specify a GPU instance
	// type, be sure to specify AL2_x86_64_GPU with the amiType parameter. If you
	// specify launchTemplate, then you can specify zero or one instance type in
	// your launch template or you can specify 0-20 instance types for instanceTypes.
	// If however, you specify an instance type in your launch template and specify
	// any instanceTypes, the node group deployment will fail. If you don't specify
	// an instance type in a launch template or for instanceTypes, then t3.medium
	// is used, by default. If you specify Spot for capacityType, then we recommend
	// specifying multiple values for instanceTypes. For more information, see Managed
	// node group capacity types (https://docs.aws.amazon.com/eks/latest/userguide/managed-node-groups.html#managed-node-group-capacity-types)
	// and Launch template support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	InstanceTypes []string `json:"instanceTypes,omitempty"`

	// The Kubernetes labels to be applied to the nodes in the node group when they
	// are created.
	Labels map[string]string `json:"labels,omitempty"`

	// The metadata applied to the node group to assist with categorization and
	// organization. Each tag consists of a key and an optional value. You define
	// both. Node group tags do not propagate to any other resources associated
	// with the node group, such as the Amazon EC2 instances or subnets.
	Tags map[string]string `json:"tags,omitempty"`

	// The scaling configuration details for the Auto Scaling group that is created
	// for your node group.
	ScalingConfig EKSNodegroupScalingConfig `json:"scalingConfig,omitempty"`

	// The Kubernetes version to use for your managed nodes. By default, the Kubernetes
	// version of the cluster is used, and this is the only accepted specified value.
	// If you specify launchTemplate, and your launch template uses a custom AMI,
	// then don't specify version, or the node group deployment will fail. For more
	// information about using launch templates with Amazon EKS, see Launch template
	// support (https://docs.aws.amazon.com/eks/latest/userguide/launch-templates.html)
	// in the Amazon EKS User Guide.
	Version string `json:"version,omitempty"`
}

type EKSNodegroupScalingConfig struct {
	// The current number of nodes that the managed node group should maintain.
	DesiredSize int32 `json:"desiredSize,omitempty"`

	// The maximum number of nodes that the managed node group can scale out to.
	// For information about the maximum number that you can specify, see Amazon
	// EKS service quotas (https://docs.aws.amazon.com/eks/latest/userguide/service-quotas.html)
	// in the Amazon EKS User Guide.
	MaxSize int32 `json:"maxSize,omitempty"`

	// The minimum number of nodes that the managed node group can scale in to.
	// This number must be greater than zero.
	MinSize int32 `json:"minSize,omitempty"`
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

type AKSMDPhase struct {
	// ProvisioningState - Defines values for AKS node pool provisioning state.
	ProvisioningState AKSProvisioningState `json:"provisioningState"`
	// PowerState - Defines values for AKS node pool power state.
	PowerState AKSPowerState `json:"powerState"`
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
	ScalingConfig AKSNodegroupScalingConfig `json:"scalingConfig,omitempty"`
	// The OSDiskSize for Agent agentpool cannot be less than 30GB or larger than 2048GB.
	OsDiskSizeGB int32 `json:"osDiskSizeGB,omitempty"`
}

type AKSNodegroupScalingConfig struct {
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

	// CreateTime: [Output only] The time the cluster was created, in
	// RFC3339 (https://www.ietf.org/rfc/rfc3339.txt) text format.
	CreateTime string `json:"createTime,omitempty"`

	// ReleaseChannel: channel specifies which release channel the cluster is
	// subscribed to.
	//
	// Possible values:
	//   "UNSPECIFIED" - No channel specified.
	//   "RAPID" - RAPID channel is offered on an early access basis for
	// customers who want to test new releases. WARNING: Versions available
	// in the RAPID Channel may be subject to unresolved issues with no
	// known workaround and are not subject to any SLAs.
	//   "REGULAR" - Clusters subscribed to REGULAR receive versions that
	// are considered GA quality. REGULAR is intended for production users
	// who want to take advantage of new features.
	//   "STABLE" - Clusters subscribed to STABLE receive versions that are
	// known to be stable and reliable in production.
	ReleaseChannel string `json:"releaseChannel,omitempty"`

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

// VirtualMachineInstancetypeList represents a list of VirtualMachineInstancetype.
// VirtualMachineInstancetype are divided into 2 categories: "custom" or "kubermatic".
// swagger:model VirtualMachineInstancetypeList
type VirtualMachineInstancetypeList struct {
	Instancetypes map[VirtualMachineInstancetypeCategory][]VirtualMachineInstancetype `json:"instancetypes,omitempty"`
}

// VirtualMachineInstancetypeCategory defines a category of VirtualMachineInstancetype.
type VirtualMachineInstancetypeCategory string

const (

	// cluster-wide resources (KubeVirt VirtualMachineClusterInstancetype).
	InstancetypeCustom VirtualMachineInstancetypeCategory = "custom"

	// namespaced resources (KubeVirt VirtualMachineInstancetype).
	InstancetypeKubermatic VirtualMachineInstancetypeCategory = "kubermatic"
)

// VirtualMachineInstanctype represents a KubeVirt VirtualMachineInstanctype
// swagger:model VirtualMachineInstancetype
type VirtualMachineInstancetype struct {
	Name string `json:"name,omitempty"`
	// Spec contains the kvinstancetypealpha1v1.VirtualMachineInstanctype.Spec object marshalled
	// Required by UI to not embed the whole kubevirt.io API object, but a marshalled spec.
	Spec string `json:"spec,omitempty"`
}

// VirtualMachinePreference represents a KubeVirt VirtualMachinePreference
// swagger:model VirtualMachinePreference
type VirtualMachinePreference VirtualMachineInstancetype

// VirtualMachinePreferenceList represents a list of VirtualMachinePreference.
// VirtualMachinePreference are divided into 2 categories: "custom" or "kubermatic".
// swagger:model VirtualMachinePreferenceList
type VirtualMachinePreferenceList struct {
	Preferences map[VirtualMachineInstancetypeCategory][]VirtualMachinePreference `json:"preferences,omitempty"`
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

// NetworkDefaults contains cluster network default settings.
// swagger:model NetworkDefaults
type NetworkDefaults struct {
	// IPv4 contains cluster network default settings for IPv4 network family.
	IPv4 *NetworkDefaultsIPFamily `json:"ipv4,omitempty"`
	// IPv6 contains cluster network default settings for IPv6 network family.
	IPv6 *NetworkDefaultsIPFamily `json:"ipv6,omitempty"`
	// ProxyMode defines the default kube-proxy mode ("ipvs" / "iptables" / "ebpf").
	ProxyMode string `json:"proxyMode,omitempty"`
	// NodeLocalDNSCacheEnabled controls whether the NodeLocal DNS Cache feature is enabled.
	NodeLocalDNSCacheEnabled bool `json:"nodeLocalDNSCacheEnabled,omitempty"`
}

// NetworkDefaultsIPFamily contains cluster network default settings for an IP family.
// swagger:model NetworkDefaultsIPFamily
type NetworkDefaultsIPFamily struct {
	// PodsCIDR contains the default network range from which POD networks are allocated.
	PodsCIDR string `json:"podsCidr,omitempty"`
	// ServicesCIDR contains the default network range from which service VIPs are allocated.
	ServicesCIDR string `json:"servicesCidr,omitempty"`
	// NodeCIDRMaskSize contains the default mask size used to address the nodes within provided Pods CIDR.
	NodeCIDRMaskSize int32 `json:"nodeCidrMaskSize,omitempty"`
	// NodePortsAllowedIPRange defines the default IP range from which access to NodePort services is allowed for applicable cloud providers.
	NodePortsAllowedIPRange string `json:"nodePortsAllowedIPRange,omitempty"`
}

// OpenstackSubnetPool is the object representing a openstack subnet pool.
// swagger:model OpenstackSubnetPool
type OpenstackSubnetPool struct {
	// Id uniquely identifies the subnet pool
	ID string `json:"id"`
	// Name is the name of the subnet pool
	Name string `json:"name"`
	// IPversion is the IP protocol version (4 or 6)
	IPversion int `json:"ipVersion"`
	// IsDefault indicates if the subnetpool is default pool or not
	IsDefault bool `json:"isDefault"`
	// Prefixes is the list of subnet prefixes
	Prefixes []string `json:"prefixes"`
}

// swagger:model ResourceQuota
type ResourceQuota struct {
	Name        string `json:"name"`
	SubjectName string `json:"subjectName"`
	SubjectKind string `json:"subjectKind"`
	// SubjectHumanReadableName contains the human-readable name for the subject(if applicable). Just filled as information in get/list.
	SubjectHumanReadableName string              `json:"subjectHumanReadableName,omitempty"`
	Quota                    Quota               `json:"quota"`
	Status                   ResourceQuotaStatus `json:"status"`
}

// swagger:model ResourceQuotaStatus
type ResourceQuotaStatus struct {
	// GlobalUsage is holds the current usage of resources for all seeds.
	GlobalUsage Quota `json:"globalUsage,omitempty"`
	// LocalUsage is holds the current usage of resources for the local seed.
	LocalUsage Quota `json:"localUsage,omitempty"`
}

// swagger:model Quota
type Quota struct {
	// CPU holds the quantity of CPU.
	CPU *int64 `json:"cpu,omitempty"`
	// Memory represents the RAM amount. Denoted in GB, rounded to 2 decimal places.
	Memory *float64 `json:"memory,omitempty"`
	// Storage represents the disk size. Denoted in GB, rounded to 2 decimal places.
	Storage *float64 `json:"storage,omitempty"`
}

// swagger:model GroupProjectBinding
type GroupProjectBinding struct {
	Name      string `json:"name"`
	Group     string `json:"group"`
	ProjectID string `json:"projectID"`
	Role      string `json:"role"`
}

// ApplicationInstallation is the object representing an ApplicationInstallation.
// swagger:model ApplicationInstallation
type ApplicationInstallation struct {
	apiv1.ObjectMeta

	Namespace string `json:"namespace,omitempty"`

	Spec *ApplicationInstallationSpec `json:"spec"`

	Status *ApplicationInstallationStatus `json:"status"`
}

// ApplicationInstallationListItem is the object representing an ApplicationInstallationListItem.
// swagger:model ApplicationInstallationListItem
type ApplicationInstallationListItem struct {
	Name string `json:"name"`

	CreationTimestamp apiv1.Time `json:"creationTimestamp,omitempty"`

	Spec *ApplicationInstallationListItemSpec `json:"spec"`

	Status *ApplicationInstallationListItemStatus `json:"status"`
}

// ApplicationInstallationListItemSpec is the object representing an ApplicationInstallationListItemSpec.
// swagger:model ApplicationInstallationListItemSpec
type ApplicationInstallationListItemSpec struct {
	// Namespace describe the desired state of the namespace where application will be created.
	Namespace NamespaceSpec `json:"namespace"`

	// ApplicationRef is a reference to identify which Application should be deployed
	ApplicationRef appskubermaticv1.ApplicationRef `json:"applicationRef"`
}

type ApplicationInstallationListItemStatus struct {
	// Conditions contains conditions an installation is in, its primary use case is status signaling between controllers or between controllers and the API
	Conditions []ApplicationInstallationCondition `json:"conditions,omitempty"`

	// ApplicationVersion contains information installing / removing application
	ApplicationVersion *appskubermaticv1.ApplicationVersion `json:"applicationVersion,omitempty"`

	// Method used to install the application
	Method appskubermaticv1.TemplateMethod `json:"method"`
}

// ApplicationInstallationBody is the object representing the POST/PUT payload of an ApplicationInstallation
// swagger:model ApplicationInstallationBody
type ApplicationInstallationBody struct {
	apiv1.ObjectMeta

	Namespace string `json:"namespace,omitempty"`

	Spec *ApplicationInstallationSpec `json:"spec"`
}

type ApplicationInstallationSpec struct {
	// Namespace describe the desired state of the namespace where application will be created.
	Namespace NamespaceSpec `json:"namespace"`

	// ApplicationRef is a reference to identify which Application should be deployed
	ApplicationRef appskubermaticv1.ApplicationRef `json:"applicationRef"`

	// Values describe overrides for manifest-rendering. It's a free yaml field.
	// +kubebuilder:pruning:PreserveUnknownFields
	Values runtime.RawExtension `json:"values,omitempty"`
	// As kubebuilder does not support interface{} as a type, deferring json decoding, seems to be our best option (see https://github.com/kubernetes-sigs/controller-tools/issues/294#issuecomment-518379253)
}

// NamespaceSpec describe the desired state of the namespace where application will be created.
type NamespaceSpec struct {
	// Name is the namespace to deploy the Application into.
	// Should be a valid lowercase RFC1123 domain name
	// +kubebuilder:validation:Pattern:=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Type=string
	Name string `json:"name"`

	// +kubebuilder:default:=true

	// Create defines whether the namespace should be created if it does not exist. Defaults to true
	Create bool `json:"create"`

	// Labels of the namespace
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations of the namespace
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ApplicationInstallationStatus is the object representing the status of an Application.
// swagger:model ApplicationInstallationStatus
// it is needed because metav1.Time used by appsv1 confuses swaggers with apiv1.Time.
type ApplicationInstallationStatus struct {
	// Conditions contains conditions an installation is in, its primary use case is status signaling between controllers or between controllers and the API
	Conditions []ApplicationInstallationCondition `json:"conditions,omitempty"`

	// ApplicationVersion contains information installing / removing application
	ApplicationVersion *appskubermaticv1.ApplicationVersion `json:"applicationVersion,omitempty"`

	// Method used to install the application
	Method appskubermaticv1.TemplateMethod `json:"method"`
}

type ApplicationInstallationCondition struct {
	// Type of ApplicationInstallation condition.
	Type appskubermaticv1.ApplicationInstallationConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time we got an update on a given condition.
	// +optional
	// swagger:strfmt date-time
	LastHeartbeatTime apiv1.Time `json:"lastHeartbeatTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	// swagger:strfmt date-time
	LastTransitionTime apiv1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
}

// swagger:model IPAMPool
type IPAMPool struct {
	Name        string                                `json:"name"`
	Datacenters map[string]IPAMPoolDatacenterSettings `json:"datacenters"`
}

// swagger:model IPAMPoolDatacenterSettings
type IPAMPoolDatacenterSettings struct {
	Type             kubermaticv1.IPAMPoolAllocationType `json:"type"`
	PoolCIDR         kubermaticv1.SubnetCIDR             `json:"poolCidr"`
	AllocationPrefix int                                 `json:"allocationPrefix,omitempty"`
	AllocationRange  int                                 `json:"allocationRange,omitempty"`
}

// ApplicationDefinition is the object representing an ApplicationDefinition.
// swagger:model ApplicationDefinition
type ApplicationDefinition struct {
	apiv1.ObjectMeta

	Spec *appskubermaticv1.ApplicationDefinitionSpec `json:"spec"`
}

// swagger:model OperatingSystemProfile
type OperatingSystemProfile struct {
	Name                    string   `json:"name"`
	OperatingSystem         string   `json:"operatingSystem"`
	SupportedCloudProviders []string `json:"supportedCloudProviders,omitempty"`
}

// ClusterServiceAccount represent a k8s service account to access cluster.
// swagger:model ClusterServiceAccount
type ClusterServiceAccount struct {
	apiv1.ObjectMeta `json:",inline"`
	Namespace        string `json:"namespace,omitempty"`
}

// ApplicationDefinitionListItem is the object representing an ApplicationDefinitionListItem.
// swagger:model ApplicationDefinitionListItem
type ApplicationDefinitionListItem struct {
	Name string `json:"name"`

	Spec ApplicationDefinitionListItemSpec `json:"spec"`
}

// ApplicationDefinitionListItemSpec defines the desired state of ApplicationDefinitionListItemSpec.
type ApplicationDefinitionListItemSpec struct {
	// Description of the application. what is its purpose
	Description string `json:"description"`
}
