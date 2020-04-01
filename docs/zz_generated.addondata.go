// TemplateData is injected into templates.
type TemplateData struct {
	SeedName       string
	DatacenterName string
	Cluster        ClusterData
	Addon          AddonData
	Credentials    Credentials
	Variables      map[string]interface{}
}

// ClusterData contains data related to the user cluster
// the addon is rendered for.
type ClusterData struct {
	Name                 string
	HumanReadableName    string
	Namespace            string
	OwnerName            string
	OwnerEmail           string
	Kubeconfig           string
	ApiserverExternalURL string
	ApiserverInternalURL string
	AdminToken           string
	CloudProviderName    string
	Version              *semver.Version
	MajorMinorVersion    string
	Network              ClusterNetwork
	Features             sets.String
}

type ClusterNetwork struct {
	DNSClusterIP      string
	DNSResolverIP     string
	PodCIDRBlocks     []string
	ServiceCIDRBlocks []string
	ProxyMode         string
}

type AddonData struct {
	Name      string
	IsDefault bool
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
