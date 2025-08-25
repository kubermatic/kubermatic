// TemplateData is the root context injected into each application values.
type TemplateData struct {
	Cluster ClusterData
}

// ClusterData contains data related to the user cluster
// the application is rendered for.
type ClusterData struct {
	Name string
	// HumanReadableName is the user-specified cluster name.
	HumanReadableName string
	// OwnerEmail is the owner's e-mail address.
	OwnerEmail string
	// ClusterAddress stores access and address information of a cluster.
	Address kubermaticv1.ClusterAddress
	// Version is the exact current cluster version.
	Version string
	// MajorMinorVersion is a shortcut for common testing on "Major.Minor" on the
	// current cluster version.
	MajorMinorVersion string
	// AutoscalerVersion is the tag which should be used for the cluster autoscaler
	AutoscalerVersion string
	// Annotations holds arbitrary non-identifying metadata attached to the cluster.
	// Transferred from the Kubermatic cluster object.
	Annotations map[string]string
	// Labels are key-value pairs used to organize, categorize, and select clusters.
	// Transferred from the Kubermatic cluster object.
	Labels map[string]string
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	// URL under which the Apiserver is available
	// +optional
	URL string `json:"url"`
	// Port is the port the API server listens on
	// +optional
	Port int32 `json:"port"`
	// ExternalName is the DNS name for this cluster
	// +optional
	ExternalName string `json:"externalName"`
	// InternalName is the seed cluster internal absolute DNS name to the API server
	// +optional
	InternalName string `json:"internalURL"`
	// AdminToken is the token for the kubeconfig, the user can download
	// +optional
	AdminToken string `json:"adminToken"`
	// IP is the external IP under which the apiserver is available
	// +optional
	IP string `json:"ip"`
	// APIServerExternalAddress is the external address of the API server (IP or DNS name)
	// This field is populated only when the API server service is of type LoadBalancer. If set, this address will be used in the
	// kubeconfig for the user cluster that can be downloaded from the KKP UI.
	// +optional
	APIServerExternalAddress string `json:"apiServerExternalAddress,omitempty"`
}
