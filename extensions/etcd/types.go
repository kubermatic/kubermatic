package etcd

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	// GroupName is the group for all our extension
	GroupName string = "etcd.coreos.com"
	// Version is the version of our extensions
	Version string = "v1beta1"
)

const (
	// TPRKind is the names of the TPR storing etcd cluster from the etcd operator
	TPRKind = "clusters"
)

var (
	// SchemeGroupVersion is the combination of group name and version for the kubernetes client
	SchemeGroupVersion = schema.GroupVersion{Group: GroupName, Version: Version}
	// SchemeBuilder provides scheme information about our extensions
	SchemeBuilder = runtime.NewSchemeBuilder(addTypes)
)

func addTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&Cluster{},
		&ClusterList{},
		&apiv1.ListOptions{},
		&metav1.ListOptions{},
	)
	return nil
}

// Cluster represent an etcd cluster
type Cluster struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata"`
	Spec            ClusterSpec       `json:"spec"`
	Status          ClusterStatus     `json:"status"`
}

// ClusterList is a list of etcd clusters.
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata"`
	// Items is a list of third party objects
	Items []Cluster `json:"items"`
}

// GetObjectKind returns the object typemeta information
func (e *Cluster) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}

// GetObjectMeta returns the object metadata
func (e *Cluster) GetObjectMeta() metav1.Object {
	return &e.Metadata
}

// GetObjectKind returns the object typemeta information
func (el *ClusterList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}

// GetListMeta returns the list object metadata
func (el *ClusterList) GetListMeta() metav1.List {
	return &el.Metadata
}

// ClusterSpec defines the spec for an etcd cluster
type ClusterSpec struct {
	// Size is the expected size of the etcd cluster.
	// The etcd-operator will eventually make the size of the running
	// cluster equal to the expected size.
	// The vaild range of the size is from 1 to 7.
	Size int `json:"size"`

	// Version is the expected version of the etcd cluster.
	// The etcd-operator will eventually make the etcd cluster version
	// equal to the expected version.
	//
	// The version must follow the [semver]( http://semver.org) format, for example "3.1.2".
	// Only etcd released versions are supported: https://github.com/coreos/etcd/releases
	//
	// If version is not set, default is "3.1.2".
	Version string `json:"version"`

	// Paused is to pause the control of the operator for the etcd cluster.
	Paused bool `json:"paused,omitempty"`

	// Pod defines the policy to create pod for the etcd container.
	Pod *PodPolicy `json:"pod,omitempty"`

	// Backup defines the policy to backup data of etcd cluster if not nil.
	// If backup policy is set but restore policy not, and if a previous backup exists,
	// this cluster would face conflict and fail to start.
	Backup *BackupPolicy `json:"backup,omitempty"`

	// Restore defines the policy to restore cluster form existing backup if not nil.
	// It's not allowed if restore policy is set and backup policy not.
	Restore *RestorePolicy `json:"restore,omitempty"`

	// SelfHosted determines if the etcd cluster is used for a self-hosted
	// Kubernetes cluster.
	SelfHosted *SelfHostedPolicy `json:"selfHosted,omitempty"`

	// etcd cluster TLS configuration
	TLS *TLSPolicy `json:"TLS"`
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
	// NodeSelector specifies a map of key-value pairs. For the pod to be eligible
	// to run on a node, the node must have each of the indicated key-value pairs as
	// labels.
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// AntiAffinity determines if the etcd-operator tries to avoid putting
	// the etcd members in the same cluster onto the same node.
	AntiAffinity bool `json:"antiAffinity"`

	// Resources is the resource requirements for the etcd container.
	// This field cannot be updated once the cluster is created.
	Resources apiv1.ResourceRequirements `json:"resources"`
}

// ClusterPhase defines the phases of a etcd-cluster
type ClusterPhase string

// ClusterPhase
const (
	ClusterPhaseNone     ClusterPhase = ""
	ClusterPhaseCreating              = "Creating"
	ClusterPhaseRunning               = "Running"
	ClusterPhaseFailed                = "Failed"
)

// ClusterCondition defines the condition of a etcd-cluster
type ClusterCondition struct {
	Type ClusterConditionType `json:"type"`

	Reason string `json:"reason"`

	TransitionTime time.Time `json:"transitionTime"`
}

// ClusterConditionType defines the condition of a etcd-cluster
type ClusterConditionType string

// ClusterConditionTypes
const (
	ClusterConditionReady = "Ready"

	ClusterConditionRemovingDeadMember = "RemovingDeadMember"

	ClusterConditionRecovering = "Recovering"

	ClusterConditionScalingUp   = "ScalingUp"
	ClusterConditionScalingDown = "ScalingDown"

	ClusterConditionUpgrading = "Upgrading"
)

// ClusterStatus defines the status for an etcd cluster
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

// MembersStatus defines the member status for an etcd cluster
type MembersStatus struct {
	// Ready are the etcd members that are ready to serve requests
	// The member names are the same as the etcd pod names
	Ready []string `json:"ready,omitempty"`
	// Unready are the etcd members not ready to serve requests
	Unready []string `json:"unready,omitempty"`
}

// BackupServiceStatus defines the backup service status for an etcd cluster
type BackupServiceStatus struct {
	// RecentBackup is status of the most recent backup created by
	// the backup service
	RecentBackup *BackupStatus `json:"recentBackup,omitempty"`

	// Backups is the totoal number of existing backups
	Backups int `json:"backups"`

	// BackupSize is the total size of existing backups in MB.
	BackupSize float64 `json:"backupSize"`
}

// BackupStatus defines the backup status for an etcd cluster
type BackupStatus struct {
	// Creation time of the backup.
	CreationTime string `json:"creationTime"`

	// Size is the size of the backup in MB.
	Size float64 `json:"size"`

	// Version is the version of the backup cluster.
	Version string `json:"version"`

	// TimeTookInSecond is the total time took to create the backup.
	TimeTookInSecond int `json:"timeTookInSecond"`
}

// BackupStorageType defines the backup storage type for an etcd cluster
type BackupStorageType string

// BackupStorageTypes
const (
	BackupStorageTypeDefault          = ""
	BackupStorageTypePersistentVolume = "PersistentVolume"
	BackupStorageTypeS3               = "S3"
)

// TLSPolicy defines the TLS policy of an etcd cluster
type TLSPolicy struct {
	// StaticTLS enables user to generate static x509 certificates and keys,
	// put them into Kubernetes secrets, and specify them into here.
	Static *StaticTLS `json:"static"`
}

// StaticTLS defines the static TLS for an etcd cluster
type StaticTLS struct {
	// ServerSecretName contains peer-interface and client-interface server x509 key/cert, along with peer and client CA cert.
	ServerSecretName string `json:"serverSecretName"`
	// ClientSecretName contains etcd client key/cert, along with client CA cert.
	ClientSecretName string `json:"clientSecretName"`
}

// BackupPolicy defines the backup policy for an etcd cluster
type BackupPolicy struct {
	// StorageType specifies the type of storage device to store backup files.
	// If it's not set by user, the default is "PersistentVolume".
	StorageType BackupStorageType `json:"storageType"`

	StorageSource `json:",inline"`

	// BackupIntervalInSecond specifies the interval between two backups.
	// The default interval is 1800 seconds.
	BackupIntervalInSecond int `json:"backupIntervalInSecond"`

	// MaxBackups is the maximum number of backup files to retain. 0 is disable backup.
	// If backup is disabled, the etcd cluster cannot recover from a
	// disaster failure (lose more than half of its members at the same
	// time).
	MaxBackups int `json:"maxBackups"`

	// CleanupBackupsOnClusterDelete tells whether to cleanup backup data if cluster is deleted.
	// By default, operator will keep the backup data.
	CleanupBackupsOnClusterDelete bool `json:"cleanupBackupsOnClusterDelete"`
}

// StorageSource defines the storage for an etcd cluster
type StorageSource struct {
	PV *PVSource `json:"pv,omitempty"`
	S3 *S3Source `json:"s3,omitempty"`
}

// PVSource defines the PV for an etcd cluster
type PVSource struct {
	// VolumeSizeInMB specifies the required volume size to perform backups.
	// Operator will claim the required size before creating the etcd cluster for backup
	// purpose.
	// If the snapshot size is larger than the size specified, backup fails.
	VolumeSizeInMB int `json:"volumeSizeInMB"`
}

// S3Source defines the S3
type S3Source struct {
}

// SelfHostedPolicy defines the SelfHostedPolicy
type SelfHostedPolicy struct {
	// BootMemberClientEndpoint specifies a bootstrap member for the cluster.
	// If there is no bootstrap member, a completely new cluster will be created.
	// The boot member will be removed from the cluster once the self-hosted cluster
	// setup successfully.
	BootMemberClientEndpoint string `json:"bootMemberClientEndpoint"`
}
