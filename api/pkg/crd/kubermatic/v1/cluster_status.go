package v1

import (
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterConditionType string

const (
	ClusterConditionControllerUpdateInProgress ClusterConditionType = "ClusterControllerUpdateInProgress"

	ClusterUpdateInProgressReason = "Current Cluster is updating its resources"
)

type ClusterCondition struct {
	// Type of cluster condition.
	Type ClusterConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// KubermaticVersion is the current kubermatic version in a cluster.
	KubermaticVersion string `json:"kubermatic_version"`
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
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

func newClusterCondition(condType ClusterConditionType, status corev1.ConditionStatus, reason, message string) *ClusterCondition {
	now := metav1.Time{
		Time: time.Now(),
	}

	return &ClusterCondition{
		Type:               condType,
		Status:             status,
		KubermaticVersion:  resources.KUBERMATICCOMMIT,
		LastHeartbeatTime:  now,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
	}
}

// ClusterStatus stores status information about a cluster.
type ClusterStatus struct {
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
	// ExtendedHealth exposes information about the current health state.
	// Extends standard health status for new states.
	ExtendedHealth ExtendedClusterHealth `json:"extendedHealth,omitempty"`

	// Deprecated
	RootCA *KeyCert `json:"rootCA,omitempty"`
	// Deprecated
	ApiserverCert *KeyCert `json:"apiserverCert,omitempty"`
	// Deprecated
	KubeletCert *KeyCert `json:"kubeletCert,omitempty"`
	// Deprecated
	ApiserverSSHKey *RSAKeys `json:"apiserverSshKey,omitempty"`
	// Deprecated
	ServiceAccountKey Bytes `json:"serviceAccountKey,omitempty"`
	// NamespaceName defines the namespace the control plane of this cluster is deployed in
	NamespaceName string `json:"namespaceName"`

	// UserName contains the name of the owner of this cluster
	UserName string `json:"userName,omitempty"`
	// UserEmail contains the email of the owner of this cluster
	UserEmail string `json:"userEmail"`

	// ErrorReason contains a error reason in case the controller encountered an error. Will be reset if the error was resolved
	ErrorReason *ClusterStatusError `json:"errorReason,omitempty"`
	// ErrorMessage contains a defauled error message in case the controller encountered an error. Will be reset if the error was resolved
	ErrorMessage *string `json:"errorMessage,omitempty"`

	// Conditions contains conditions the cluster is in, its primary use case is status signaling between controllers or between
	// controllers and the API
	Conditions []ClusterCondition `json:"conditions,omitempty"`

	// CloudMigrationRevision describes the latest version of the migration that has been done
	// It is used to avoid redundant and potentially costly migrations
	CloudMigrationRevision int `json:"cloudMigrationRevision"`
}

func (cs *ClusterStatus) SetClusterUpdateInProgressConditionTrue(message string) {
	condition := newClusterCondition(ClusterConditionControllerUpdateInProgress, corev1.ConditionTrue,
		ClusterUpdateInProgressReason, message)
	cs.setClusterCondition(*condition)
}

func (cs *ClusterStatus) setClusterCondition(c ClusterCondition) {
	pos, clusterCondition := cs.getClusterCondition(c.Type)
	if clusterCondition != nil &&
		clusterCondition.Status == c.Status && clusterCondition.Reason == c.Reason && clusterCondition.Message == c.Message {
		return
	}

	if clusterCondition != nil {
		cs.Conditions[pos] = c
	} else {
		cs.Conditions = append(cs.Conditions, c)
	}
}

func (cs *ClusterStatus) getClusterCondition(conditionType ClusterConditionType) (int, *ClusterCondition) {
	for i, c := range cs.Conditions {
		if conditionType == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

func (cs *ClusterStatus) ClearCondition(conditionType ClusterConditionType) {
	pos, _ := cs.getClusterCondition(conditionType)
	if pos == -1 {
		return
	}
	cs.Conditions = append(cs.Conditions[:pos], cs.Conditions[pos+1:]...)
}

type ClusterStatusError string

const (
	InvalidConfigurationClusterError ClusterStatusError = "InvalidConfiguration"
	UnsupportedChangeClusterError    ClusterStatusError = "UnsupportedChange"
	ReconcileClusterError            ClusterStatusError = "ReconcileError"
)
