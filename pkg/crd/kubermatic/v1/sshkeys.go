package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SSHKeyResourceName represents "Resource" defined in Kubernetes
	SSHKeyResourceName = "usersshkeies"

	// SSHKeyKind represents "Kind" defined in Kubernetes
	SSHKeyKind = "UserSSHKey"
)

//+genclient
//+genclient:nonNamespaced
//+resourceName=usersshkeies

// UserSSHKey specifies a users UserSSHKey
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UserSSHKey struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SSHKeySpec `json:"spec"`
}

type SSHKeySpec struct {
	Owner       string   `json:"owner"`
	Name        string   `json:"name"`
	Fingerprint string   `json:"fingerprint"`
	PublicKey   string   `json:"publicKey"`
	Clusters    []string `json:"clusters"`
}

func (sk *UserSSHKey) IsUsedByCluster(clustername string) bool {
	if sk.Spec.Clusters == nil {
		return false
	}
	for _, name := range sk.Spec.Clusters {
		if name == clustername {
			return true
		}
	}
	return false
}

func (sk *UserSSHKey) RemoveFromCluster(clustername string) {
	for i, cl := range sk.Spec.Clusters {
		if cl != clustername {
			continue
		}
		// Don't break we don't check for duplicates when adding clusters!
		sk.Spec.Clusters = append(sk.Spec.Clusters[:i], sk.Spec.Clusters[i+1:]...)
	}
}

func (sk *UserSSHKey) AddToCluster(clustername string) {
	sk.Spec.Clusters = append(sk.Spec.Clusters, clustername)
}

// UserSSHKeyList specifies a users UserSSHKey
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type UserSSHKeyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []UserSSHKey `json:"items"`
}
