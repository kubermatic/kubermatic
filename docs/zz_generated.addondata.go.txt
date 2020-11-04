// TemplateData is the root context injected into each addon manifest file.
type TemplateData struct {
	SeedName       string
	DatacenterName string
	Cluster        ClusterData
	Credentials    Credentials
	Variables      map[string]interface{}
}

// ClusterData contains data related to the user cluster
// the addon is rendered for.
type ClusterData struct {
	// Type is either "kubernetes" or "openshift".
	Type string
	// Name is the auto-generated, internal cluster name, e.g. "bbc8sc24wb".
	Name string
	// HumanReadableName is the user-specified cluster name.
	HumanReadableName string
	// Namespace is the full namespace for the cluster's control plane.
	Namespace string
	// OwnerName is the owner's full name.
	OwnerName string
	// OwnerEmail is the owner's e-mail address.
	OwnerEmail string
	// Labels are the labels users have configured for their cluster, including
	// system-defined labels like the project ID.
	Labels map[string]string
	// Annotations are the annotations on the cluster resource, usually
	// cloud-provider related information like regions.
	Annotations map[string]string
	// Kubeconfig is a YAML-encoded kubeconfig with cluster-admin permissions
	// inside the user-cluster. The kubeconfig uses the external URL to reach
	// the apiserver.
	Kubeconfig string
	// ApiserverExternalURL is the full URL to the apiserver service from the
	// outside, including protocol and port number. It does not contain any
	// trailing slashes.
	ApiserverExternalURL string
	// ApiserverExternalURL is the full URL to the apiserver from within the
	// seed cluster itself. It does not contain any trailing slashes.
	ApiserverInternalURL string
	// AdminToken is the cluster's admin token.
	AdminToken string
	// CloudProviderName is the name of the cloud provider used, one of
	// "alibaba", "aws", "azure", "bringyourown", "digitalocean", "gcp",
	// "hetzner", "kubevirt", "openstack", "packet", "vsphere" depending on
	// the configured datacenters.
	CloudProviderName string
	// Version is the exact cluster version.
	Version *semver.Version
	// MajorMinorVersion is a shortcut for common testing on "Major.Minor".
	MajorMinorVersion string
	// Network contains DNS and CIDR settings for the cluster.
	Network ClusterNetwork
	// Features is a set of enabled features for this cluster.
	Features sets.String
}

type ClusterNetwork struct {
	DNSClusterIP      string
	DNSResolverIP     string
	PodCIDRBlocks     []string
	ServiceCIDRBlocks []string
	ProxyMode         string
}

type Credentials struct {
	AWS          AWSCredentials
	Azure        AzureCredentials
	Digitalocean DigitaloceanCredentials
	GCP          GCPCredentials
	Hetzner      HetznerCredentials
	Openstack    OpenstackCredentials
	Packet       PacketCredentials
	Kubevirt     KubevirtCredentials
	VSphere      VSphereCredentials
	Alibaba      AlibabaCredentials
	Anexia       AnexiaCredentials
}

type AWSCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

type AzureCredentials struct {
	TenantID       string
	SubscriptionID string
	ClientID       string
	ClientSecret   string
}

type DigitaloceanCredentials struct {
	Token string
}

type GCPCredentials struct {
	ServiceAccount string
}

type HetznerCredentials struct {
	Token string
}

type OpenstackCredentials struct {
	Username string
	Password string
	Tenant   string
	TenantID string
	Domain   string
}

type PacketCredentials struct {
	APIKey    string
	ProjectID string
}

type KubevirtCredentials struct {
	KubeConfig string
}

type VSphereCredentials struct {
	Username string
	Password string
}

type AlibabaCredentials struct {
	AccessKeyID     string
	AccessKeySecret string
}

type AnexiaCredentials struct {
	Token string
}
