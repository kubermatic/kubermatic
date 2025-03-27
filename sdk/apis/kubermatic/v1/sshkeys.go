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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// SSHKeyResourceName represents "Resource" defined in Kubernetes.
	SSHKeyResourceName = "usersshkeys"

	// SSHKeyKind represents "Kind" defined in Kubernetes.
	SSHKeyKind = "UserSSHKey"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".spec.name",name="HumanReadableName",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.owner",name="Owner",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.project",name="Project",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.fingerprint",name="Fingerprint",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// UserSSHKey specifies a users UserSSHKey.
type UserSSHKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SSHKeySpec `json:"spec,omitempty"`
}

type SSHKeySpec struct {
	// Name is the human readable name for this SSH key.
	Name string `json:"name"`
	// Owner is the name of the User object that owns this SSH key.
	// Deprecated: This field is not used anymore.
	// +optional
	Owner string `json:"owner,omitempty"`
	// Project is the name of the Project object that this SSH key belongs to.
	// This field is immutable.
	Project string `json:"project"`
	// Clusters is the list of cluster names that this SSH key is assigned to.
	Clusters []string `json:"clusters"`
	// Fingerprint is calculated server-side based on the supplied public key
	// and doesn't need to be set by clients.
	// +optional
	Fingerprint string `json:"fingerprint"`
	// PublicKey is the SSH public key.
	PublicKey string `json:"publicKey"`
}

func (sk *UserSSHKey) IsUsedByCluster(clustername string) bool {
	return sets.New(sk.Spec.Clusters...).Has(clustername)
}

func (sk *UserSSHKey) RemoveFromCluster(clustername string) {
	sk.Spec.Clusters = sets.List(sets.New(sk.Spec.Clusters...).Delete(clustername))
}

func (sk *UserSSHKey) AddToCluster(clustername string) {
	sk.Spec.Clusters = sets.List(sets.New(sk.Spec.Clusters...).Insert(clustername))
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// UserSSHKeyList specifies a users UserSSHKey.
type UserSSHKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []UserSSHKey `json:"items"`
}
