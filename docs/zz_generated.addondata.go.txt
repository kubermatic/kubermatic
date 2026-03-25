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
	// Type is only "kubernetes"
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

	// ClusterAddress stores access and address information of a cluster.
	Address kubermaticv1.ClusterAddress

	// CloudProviderName is the name of the cloud provider used, one of
	// "alibaba", "aws", "azure", "bringyourown", "digitalocean", "gcp",
	// "hetzner", "kubevirt", "openstack", "vsphere" depending on
	// the configured datacenters.
	CloudProviderName string
	// Version is the exact current cluster version.
	Version *semverlib.Version
	// MajorMinorVersion is a shortcut for common testing on "Major.Minor" on the
	// current cluster version.
	MajorMinorVersion string
	// Network contains DNS and CIDR settings for the cluster.
	Network ClusterNetwork
	// Features is a set of enabled features for this cluster.
	Features sets.Set[string]
	// CNIPlugin contains the CNIPlugin settings
	CNIPlugin CNIPlugin
	// CSI specific options, dependent on provider
	CSI CSIOptions
	// MLA contains monitoring, logging and alerting related settings for the user cluster.
	MLA MLASettings
	// CSIMigration indicates if the cluster needed the CSIMigration
	CSIMigration bool
	// KubeVirtInfraStorageClasses is a list of storage classes from KubeVirt infra cluster that are used for
	// initialization of user cluster storage classes by the CSI driver kubevirt (hot pluggable disks)
	KubeVirtInfraStorageClasses []kubermaticv1.KubeVirtInfraStorageClass
	// KubeVirtInfraVolumeSnapshotClasses is a list of volume snapshot classes from the KubeVirt infra cluster
	// that are used for initialization of user cluster volume snapshot classes by the CSI driver kubevirt.
	KubeVirtInfraVolumeSnapshotClasses []kubermaticv1.KubeVirtInfraVolumeSnapshotClass
	// DisableCSIDriver indicates if csi drivers (csi addon) is disabled for the user cluster or not.
	DisableCSIDriver bool
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

type ClusterNetwork struct {
	DNSDomain            string
	DNSClusterIP         string
	DNSResolverIP        string
	PodCIDRBlocks        []string
	ServiceCIDRBlocks    []string
	ProxyMode            string
	StrictArp            *bool
	DualStack            bool
	PodCIDRIPv4          string
	PodCIDRIPv6          string
	NodeCIDRMaskSizeIPv4 int32
	NodeCIDRMaskSizeIPv6 int32
	IPAMAllocations      map[string]IPAMAllocation
	NodePortRange        string
}

type CNIPlugin struct {
	Type    string
	Version string
}

type CSIOptions struct {

	// vsphere
	// StoragePolicy is the storage policy to use for vsphere csi addon
	StoragePolicy string

	// nutanix
	StorageContainer        string
	Fstype                  string
	SsSegmentedIscsiNetwork *bool

	// vmware Cloud Director
	StorageProfile string
	Filesystem     string

	// openstack
	CinderTopologyEnabled bool

	// kubevirt
	OverwriteRegistry string
}

type MLASettings struct {
	// MonitoringEnabled is the flag for enabling monitoring in user cluster.
	MonitoringEnabled bool
	// LoggingEnabled is the flag for enabling logging in user cluster.
	LoggingEnabled bool
}

type Credentials struct {
	AWS                 AWSCredentials
	Azure               AzureCredentials
	Baremetal           BaremetalCredentials
	Digitalocean        DigitaloceanCredentials
	GCP                 GCPCredentials
	Hetzner             HetznerCredentials
	Openstack           OpenstackCredentials
	Kubevirt            KubevirtCredentials
	VSphere             VSphereCredentials
	Alibaba             AlibabaCredentials
	Anexia              AnexiaCredentials
	Nutanix             NutanixCredentials
	VMwareCloudDirector VMwareCloudDirectorCredentials
}

type AWSCredentials struct {
	AccessKeyID          string
	SecretAccessKey      string
	AssumeRoleARN        string
	AssumeRoleExternalID string
}

type AzureCredentials struct {
	TenantID       string
	SubscriptionID string
	ClientID       string
	ClientSecret   string
}

type BaremetalCredentials struct {
	Tinkerbell TinkerbellCredentials
}

type TinkerbellCredentials struct {
	// Admin kubeconfig for Tinkerbell cluster
	Kubeconfig string
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
	Username                    string
	Password                    string
	Project                     string
	ProjectID                   string
	Domain                      string
	ApplicationCredentialID     string
	ApplicationCredentialSecret string
	Token                       string
}

type KubevirtCredentials struct {
	// Admin kubeconfig for KubeVirt cluster
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

type NutanixCredentials struct {
	Username    string
	Password    string
	CSIUsername string
	CSIPassword string
	ProxyURL    string
}

type VMwareCloudDirectorCredentials struct {
	Username     string
	Password     string
	APIToken     string
	Organization string
	VDC          string
}
