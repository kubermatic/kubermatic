/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterBackupStorageLocationKind represents "Kind" defined in Kubernetes.
	ClusterBackupStorageLocationKind = "ClusterBackupStorageLocation"
)

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cbsl
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase",description="Backup Storage Location status such as Available/Unavailable"
// +kubebuilder:printcolumn:name="Last Validated",type="date",JSONPath=".status.lastValidationTime",description="LastValidationTime is the last time the backup store location was validated"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Default",type="boolean",JSONPath=".spec.default",description="Default backup storage location"

// ClusterBackupStorageLocation is a KKP wrapper around Velero BSL spec.
type ClusterBackupStorageLocation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is a Velero BSL spec
	Spec   velerov1.BackupStorageLocationSpec   `json:"spec,omitempty"`
	Status velerov1.BackupStorageLocationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ClusterBackupStorageLocationList is a list of ClusterBackupStorageLocations.
type ClusterBackupStorageLocationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items is a list of EtcdBackupConfig objects.
	Items []ClusterBackupStorageLocation `json:"items"`
}
