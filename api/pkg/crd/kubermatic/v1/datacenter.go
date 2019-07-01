package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubermatic/machine-controller/pkg/providerconfig"
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
	// Detailed location of the cluster. For informational purposes only
	Location string `json:"location,omitempty"`
	// A reference to the Kubeconfig of this cluster
	Kubeconfig corev1.ObjectReference `json:"kubeconfig"`
	// The possible locations for the nodes
	NodeLocations map[string]NodeLocation `json:"node_location,omitempty"`
	// Optional: Overwrite the DNS domain for this seed
	SeedDNSOverwrite *string `json:"seed_dns_overwrite,omitempty"`
}

type NodeLocation struct {
	DatacenterSpec `json:",inline"`
	Node           NodeSettings `json:"node"`
}

// DatacenterSpec describes mutually points to provider datacenter spec
type DatacenterSpec struct {
	Digitalocean *DigitaloceanSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnSpec `json:"bringyourown,omitempty"`
	AWS          *AWSSpec          `json:"aws,omitempty"`
	Azure        *AzureSpec        `json:"azure,omitempty"`
	Openstack    *OpenstackSpec    `json:"openstack,omitempty"`
	Packet       *PacketSpec       `json:"packet,omitempty"`
	Hetzner      *HetznerSpec      `json:"hetzner,omitempty"`
	VSphere      *VSphereSpec      `json:"vsphere,omitempty"`
	GCP          *GCPSpec          `json:"gcp,omitempty"`
}

// ImageList defines a map of operating system and the image to use
type ImageList map[providerconfig.OperatingSystem]string

// HetznerSpec describes a Hetzner cloud datacenter
type HetznerSpec struct {
	Datacenter string `yaml:"datacenter"`
	Location   string `yaml:"location"`
}

// DigitaloceanSpec describes a DigitalOcean datacenter
type DigitaloceanSpec struct {
	Region string `yaml:"region"`
}

// OpenstackSpec describes a open stack datacenter
type OpenstackSpec struct {
	AuthURL           string `json:"auth_url"`
	AvailabilityZone  string `json:"availability_zone"`
	Region            string `json:"region"`
	IgnoreVolumeAZ    bool   `json:"ignore_volume_az"`
	EnforceFloatingIP bool   `json:"enforce_floating_ip"`
	// Used for automatic network creation
	DNSServers []string  `json:"dns_servers"`
	Images     ImageList `json:"images"`
	// Gets mapped to the "manage-security-groups" setting in the cloud config.
	// See https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#load-balancer
	// To make this change backwards compatible, this will default to true.
	ManageSecurityGroups *bool `json:"manage_security_groups"`
	// Gets mapped to the "trust-device-path" setting in the cloud config.
	// See https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#block-storage
	TrustDevicePath      *bool                         `json:"trust_device_path"`
	NodeSizeRequirements OpenstackNodeSizeRequirements `json:"node_size_requirements"`
}

type OpenstackNodeSizeRequirements struct {
	// VCPUs is the minimum required amount of (virtual) CPUs
	MinimumVCPUs int `json:"minimum_vcpus"`
	// MinimumMemory is the minimum required amount of memory, measured in MB
	MinimumMemory int `json:"minimum_memory"`
}

// AzureSpec describes an Azure cloud datacenter
type AzureSpec struct {
	Location string `json:"location"`
}

// VSphereSpec describes a vsphere datacenter
type VSphereSpec struct {
	Endpoint      string `json:"endpoint"`
	AllowInsecure bool   `json:"allow_insecure"`

	Datastore  string    `json:"datastore"`
	Datacenter string    `json:"datacenter"`
	Cluster    string    `json:"cluster"`
	RootPath   string    `json:"root_path"`
	Templates  ImageList `json:"templates"`

	// Infra management user is an optional user that will be used only
	// for everything except the cloud provider functionality which will
	// still use the credentials passed in via the frontend/api
	InfraManagementUser *VSphereCredentials `json:"infra_management_user,omitempty"`
}

// AWSSpec describes a aws datacenter
type AWSSpec struct {
	Region        string    `json:"region"`
	Images        ImageList `json:"images"`
	ZoneCharacter string    `json:"zone_character"`
}

// BringYourOwnSpec describes a datacenter our of bring your own nodes
type BringYourOwnSpec struct {
}

// PacketSpec describes a packet datacenter
type PacketSpec struct {
	Facilities []string `json:"facilities"`
}

// GCPSpec describes a GCP datacenter
type GCPSpec struct {
	Region       string   `json:"region"`
	ZoneSuffixes []string `json:"zone_suffixes"`
	Regional     bool     `json:"regional,omitempty"`
}

// NodeSettings are node specific which can be configured on datacenter level
type NodeSettings struct {
	// If set, this proxy will be configured on all nodes.
	HTTPProxy string `json:"http_proxy,omitempty"`
	// If set this will be set as NO_PROXY on the node
	NoProxy string `json:"no_proxy,omitempty"`
	// If set, this image registry will be configured as insecure on the container runtime.
	InsecureRegistries []string `json:"insecure_registries,omitempty"`
	// Translates to --pod-infra-container-image on the kubelet. If not set, the kubelet will default it
	PauseImage string `json:"pause_image,omitempty"`
	// The hyperkube image to use. Currently only Container Linux uses it
	HyperkubeImage string `json:"hyperkube_image,omitempty"`
}
