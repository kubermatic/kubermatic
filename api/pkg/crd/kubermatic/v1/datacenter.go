package v1

import (
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SeedDatacenter is the type representing a SeedDatacenter
type SeedDatacenter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SeedDatacenterSpec `json:"spec"`
}

// The spec for a seed data
type SeedDatacenterSpec struct {
	// Country of the seed. For informational purposes only
	Country string `json:"country,omitempty"`
	// Detailled location of the cluster. For informational purposes only
	Location string `json:"location,omitempty"`
	// A reference to the Kubeconfig of this cluster
	Kubeconfig corev1.ObjectReference `json:"kubeconfig"`
	// The possible locations for the nodes
	NodeLocations map[string]NodeLocation `json:node_location`
	// Optional: Overwrite the DNS domain for this seed
	SeedDNSOverwrite *string `json:"seed_dns_overwrite,omitempty"`
}

type NodeLocation struct {
	provider.DatacenterSpec `json:",inline"`
	Node                    provider.NodeSettings `json:"node"`
}
