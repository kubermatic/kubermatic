package v1beta2

import (
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type BackupStorageType string

const (
	EtcdClusterPlural = "etcdclusters"
)

const (
	BackupStorageTypeDefault          = ""
	BackupStorageTypePersistentVolume = "PersistentVolume"
	BackupStorageTypeS3               = "S3"
	BackupStorageTypeABS              = "ABS"

	AWSSecretCredentialsFileName = "credentials"
	AWSSecretConfigFileName      = "config"
)

const (
	defaultBaseImage = "quay.io/coreos/etcd"
	defaultVersion   = "3.1.8"
)

//+genclient

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EtcdCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ClusterSpec   `json:"spec"`
	Status            ClusterStatus `json:"status"`
}

type ClusterSpec struct {
	// Size is the expected size of the etcd cluster.
	// The etcd-operator will eventually make the size of the running
	// cluster equal to the expected size.
	// The vaild range of the size is from 1 to 7.
	Size int `json:"size"`

	// BaseImage is the base etcd image name that will be used to launch
	// etcd clusters. This is useful for private registries, etc.
	//
	// If image is not set, default is quay.io/coreos/etcd
	BaseImage string `json:"baseImage"`

	// Version is the expected version of the etcd cluster.
	// The etcd-operator will eventually make the etcd cluster version
	// equal to the expected version.
	//
	// The version must follow the [semver]( http://semver.org) format, for example "3.1.8".
	// Only etcd released versions are supported: https://github.com/coreos/etcd/releases
	//
	// If version is not set, default is "3.1.8".
	Version string `json:"version,omitempty"`

	// Paused is to pause the control of the operator for the etcd cluster.
	Paused bool `json:"paused,omitempty"`

	// Pod defines the policy to create pod for the etcd pod.
	//
	// Updating Pod does not take effect on any existing etcd pods.
	Pod *PodPolicy `json:"pod,omitempty"`

	// Backup defines the policy to backup data of etcd cluster if not nil.
	// If backup policy is set but restore policy not, and if a previous backup exists,
	// this cluster would face conflict and fail to start.
	Backup *BackupPolicy `json:"backup,omitempty"`

	// Restore defines the policy to restore cluster form existing backup if not nil.
	// It's not allowed if restore policy is set and backup policy not.
	//
	// Restore is a cluster initialization configuration. It cannot be updated.
	Restore *RestorePolicy `json:"restore,omitempty"`

	// SelfHosted determines if the etcd cluster is used for a self-hosted
	// Kubernetes cluster.
	//
	// SelfHosted is a cluster initialization configuration. It cannot be updated.
	SelfHosted *SelfHostedPolicy `json:"selfHosted,omitempty"`

	// etcd cluster TLS configuration
	TLS *TLSPolicy `json:"TLS,omitempty"`
}

// RestorePolicy defines the policy to restore cluster form existing backup if not nil.
type RestorePolicy struct {
	// BackupClusterName is the cluster name of the backup to recover from.
	BackupClusterName string `json:"backupClusterName"`

	// StorageType specifies the type of storage device to store backup files.
	// If not set, the default is "PersistentVolume".
	StorageType BackupStorageType `json:"storageType"`
}

// PodPolicy defines the policy to create pod for the etcd container.
type PodPolicy struct {
	// Labels specifies the labels to attach to pods the operator creates for the
	// etcd cluster.
	// "app" and "etcd_*" labels are reserved for the internal use of the etcd operator.
	// Do not overwrite them.
	Labels map[string]string `json:"labels,omitempty"`

	// NodeSelector specifies a map of key-value pairs. For the pod to be eligible
	// to run on a node, the node must have each of the indicated key-value pairs as
	// labels.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// AntiAffinity determines if the etcd-operator tries to avoid putting
	// the etcd members in the same cluster onto the same node.
	AntiAffinity bool `json:"antiAffinity,omitempty"`

	// Resources is the resource requirements for the etcd container.
	// This field cannot be updated once the cluster is created.
	Resources v1.ResourceRequirements `json:"resources,omitempty"`

	// Tolerations specifies the pod's tolerations.
	Tolerations []v1.Toleration `json:"tolerations,omitempty"`

	// List of environment variables to set in the etcd container.
	// This is used to configure etcd process. etcd cluster cannot be created, when
	// bad environement variables are provided. Do not overwrite any flags used to
	// bootstrap the cluster (for example `--initial-cluster` flag).
	// This field cannot be updated.
	EtcdEnv []v1.EnvVar `json:"etcdEnv,omitempty"`
}

type ClusterPhase string

const (
	ClusterPhaseNone     ClusterPhase = ""
	ClusterPhaseCreating              = "Creating"
	ClusterPhaseRunning               = "Running"
	ClusterPhaseFailed                = "Failed"
)

type ClusterCondition struct {
	Type ClusterConditionType `json:"type"`

	Reason string `json:"reason"`

	TransitionTime string `json:"transitionTime"`
}

type ClusterConditionType string

const (
	ClusterConditionReady = "Ready"

	ClusterConditionRemovingDeadMember = "RemovingDeadMember"

	ClusterConditionRecovering = "Recovering"

	ClusterConditionScalingUp   = "ScalingUp"
	ClusterConditionScalingDown = "ScalingDown"

	ClusterConditionUpgrading = "Upgrading"
)

type ClusterStatus struct {
	// Phase is the cluster running phase
	Phase  ClusterPhase `json:"phase"`
	Reason string       `json:"reason"`

	// ControlPuased indicates the operator pauses the control of the cluster.
	ControlPaused bool `json:"controlPaused"`

	// Condition keeps ten most recent cluster conditions
	Conditions []ClusterCondition `json:"conditions"`

	// Size is the current size of the cluster
	Size int `json:"size"`
	// Members are the etcd members in the cluster
	Members MembersStatus `json:"members"`
	// CurrentVersion is the current cluster version
	CurrentVersion string `json:"currentVersion"`
	// TargetVersion is the version the cluster upgrading to.
	// If the cluster is not upgrading, TargetVersion is empty.
	TargetVersion string `json:"targetVersion"`

	// BackupServiceStatus is the status of the backup service.
	// BackupServiceStatus only exists when backup is enabled in the
	// cluster spec.
	BackupServiceStatus *BackupServiceStatus `json:"backupServiceStatus,omitempty"`
}

type MembersStatus struct {
	// Ready are the etcd members that are ready to serve requests
	// The member names are the same as the etcd pod names
	Ready []string `json:"ready,omitempty"`
	// Unready are the etcd members not ready to serve requests
	Unready []string `json:"unready,omitempty"`
}

type BackupPolicy struct {
	// Pod defines the policy to create the backup pod.
	Pod *PodPolicy `json:"pod,omitempty"`

	// StorageType specifies the type of storage device to store backup files.
	// If it's not set by user, the default is "PersistentVolume".
	StorageType BackupStorageType `json:"storageType"`

	StorageSource `json:",inline"`

	// BackupIntervalInSecond specifies the interval between two backups.
	// The default interval is 1800 seconds.
	BackupIntervalInSecond int `json:"backupIntervalInSecond"`

	// If greater than 0, MaxBackups is the maximum number of backup files to retain.
	// If equal to 0, it means unlimited backups.
	// Otherwise, it is invalid.
	MaxBackups int `json:"maxBackups"`

	// AutoDelete tells whether to cleanup backup data if cluster is deleted.
	// By default (false), operator will keep the backup data.
	AutoDelete bool `json:"autoDelete"`
}

type StorageSource struct {
	PV *PVSource `json:"pv,omitempty"`
	S3 *S3Source `json:"s3,omitempty"`
	// ABS represents an Azure Blob Storage resource for storing etcd backups
	ABS *ABSSource `json:"abs,omitempty"`
}

type PVSource struct {
	// VolumeSizeInMB specifies the required volume size to perform backups.
	// Operator will claim the required size before creating the etcd cluster for backup
	// purpose.
	// If the snapshot size is larger than the size specified, backup fails.
	VolumeSizeInMB int `json:"volumeSizeInMB"`

	// StorageClass indicates what Kubernetes storage class will be used to
	// snapshot etcd cluster state to a persistent volume. This enables the user
	// to have fine-grained control over how backups work, since it uses the
	// existing StorageClass mechanism in Kubernetes.
	StorageClass string `json:"storageClass"`
}

// TODO: support per cluster S3 Source configuration.
type S3Source struct {
	// The name of the AWS S3 bucket to store backups in.
	//
	// S3Bucket overwrites the default etcd operator wide bucket.
	S3Bucket string `json:"s3Bucket,omitempty"`

	// Prefix is the S3 prefix used to prefix the bucket path.
	// It's the prefix at the beginning.
	// After that, it will have version and cluster specific paths.
	Prefix string `json:"prefix,omitempty"`

	// The name of the secret object that stores the AWS credential and config files.
	// The file name of the credential MUST be 'credentials'.
	// The file name of the config MUST be 'config'.
	// The profile to use in both files will be 'default'.
	//
	// AWSSecret overwrites the default etcd operator wide AWS credential and config.
	AWSSecret string `json:"awsSecret,omitempty"`
}

// ABSSource represents an Azure Blob Storage (ABS) backup storage source
type ABSSource struct {
	// ABSContainer is the name of the ABS container to store backups in.
	ABSContainer string `json:"absContainer,omitempty"`

	// ABSSecret is the name of the secret object that stores the ABS credentials.
	//
	// Within the secret object, the following fields MUST be provided:
	// 'storage-account' holding the Azure Storage account name
	// 'storage-key' holding the Azure Storage account key
	ABSSecret string `json:"absSecret,omitempty"`
}

type BackupServiceStatus struct {
	// RecentBackup is status of the most recent backup created by
	// the backup service
	RecentBackup *BackupStatus `json:"recentBackup,omitempty"`

	// Backups is the totoal number of existing backups
	Backups int `json:"backups"`

	// BackupSize is the total size of existing backups in MB.
	BackupSize float64 `json:"backupSize"`
}

type BackupStatus struct {
	// Creation time of the backup.
	CreationTime string `json:"creationTime"`

	// Size is the size of the backup in MB.
	Size float64 `json:"size"`

	// Revision is the revision of the backup.
	Revision int64 `json:"revision"`

	// Version is the version of the backup cluster.
	Version string `json:"version"`

	// TimeTookInSecond is the total time took to create the backup.
	TimeTookInSecond int `json:"timeTookInSecond"`
}

// TLSPolicy defines the TLS policy of an etcd cluster
type TLSPolicy struct {
	// StaticTLS enables user to generate static x509 certificates and keys,
	// put them into Kubernetes secrets, and specify them into here.
	Static *StaticTLS `json:"static,omitempty"`
}

type StaticTLS struct {
	// Member contains secrets containing TLS certs used by each etcd member pod.
	Member *MemberSecret `json:"member,omitempty"`
	// OperatorSecret is the secret containing TLS certs used by operator to
	// talk securely to this cluster.
	OperatorSecret string `json:"operatorSecret,omitempty"`
}

type MemberSecret struct {
	// PeerSecret is the secret containing TLS certs used by each etcd member pod
	// for the communication between etcd peers.
	PeerSecret string `json:"peerSecret,omitempty"`
	// ServerSecret is the secret containing TLS certs used by each etcd member pod
	// for the communication between etcd server and its clients.
	ServerSecret string `json:"serverSecret,omitempty"`
}

type SelfHostedPolicy struct {
	// BootMemberClientEndpoint specifies a bootstrap member for the cluster.
	// If there is no bootstrap member, a completely new cluster will be created.
	// The boot member will be removed from the cluster once the self-hosted cluster
	// setup successfully.
	BootMemberClientEndpoint string `json:"bootMemberClientEndpoint,omitempty"`

	// SkipBootMemberRemoval specifies whether the removal of the bootstrap member
	// should be skipped. By default the operator will automatically remove the
	// bootstrap member from the new cluster - this happens during the pivot
	// procedure and is the first step of decommissioning the bootstrap member.
	// If unspecified, the default is `false`. If set to `true`, you are
	// expected to remove the boot member yourself from the etcd cluster.
	SkipBootMemberRemoval bool `json:"skipBootMemberRemoval,omitempty"`
}

// EtcdClusterList is a list of etcd clusters.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type EtcdClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: http://releases.k8s.io/HEAD/docs/devel/api-conventions.md#metadata
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EtcdCluster `json:"items"`
}
