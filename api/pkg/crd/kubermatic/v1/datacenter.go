package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubermatic/machine-controller/pkg/providerconfig"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SeedDatacenterList is the type representing a SeedDatacenterList
type SeedList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of seeds
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md
	Items []Seed `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SeedDatacenter is the type representing a SeedDatacenter
type Seed struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SeedSpec `json:"spec"`
}

func (s *Seed) SetDefaults() {
	if s.Spec.ProxySettings != nil {
		for key, dc := range s.Spec.Datacenters {
			if dc.Node.HTTPProxy == nil {
				dc.Node.HTTPProxy = s.Spec.ProxySettings.HTTPProxy
			}
			if dc.Node.NoProxy == nil {
				dc.Node.NoProxy = s.Spec.ProxySettings.NoProxy
			}
			s.Spec.Datacenters[key] = dc
		}
	}
}

// The spec for a seed data
type SeedSpec struct {
	// Country of the seed. For informational purposes only
	Country string `json:"country,omitempty"`
	// Detailed location of the cluster. For informational purposes only
	Location string `json:"location,omitempty"`
	// A reference to the Kubeconfig of this cluster
	Kubeconfig corev1.ObjectReference `json:"kubeconfig"`
	// The possible datacenters for the nodes
	Datacenters map[string]Datacenter `json:"datacenters,omitempty"`
	// Optional: Overwrite the DNS domain for this seed
	SeedDNSOverwrite *string `json:"seed_dns_overwrite,omitempty"`
	// Optional: Configure a http proxy for this seed
	ProxySettings *ProxySettings `json:"proxy_settings,omitempty"`
}

type Datacenter struct {
	// Country of the seed. For informational purposes only
	Country string `json:"country,omitempty"`
	// Detailed location of the cluster. For informational purposes only
	Location string `json:"location,omitempty"`
	// Node holds node-specific settings, like e.g. HTTP proxy, insecure registries and the like
	Node NodeSettings   `json:"node"`
	Spec DatacenterSpec `json:"spec"`
}

// DatacenterSpec mutually points to provider datacenter spec
type DatacenterSpec struct {
	Digitalocean *DatacenterSpecDigitalocean `json:"digitalocean,omitempty"`
	BringYourOwn *DatacenterSpecBringYourOwn `json:"bringyourown,omitempty"`
	AWS          *DatacenterSpecAWS          `json:"aws,omitempty"`
	Azure        *DatacenterSpecAzure        `json:"azure,omitempty"`
	Openstack    *DatacenterSpecOpenstack    `json:"openstack,omitempty"`
	Packet       *DatacenterSpecPacket       `json:"packet,omitempty"`
	Hetzner      *DatacenterSpecHetzner      `json:"hetzner,omitempty"`
	VSphere      *DatacenterSpecVSphere      `json:"vsphere,omitempty"`
	GCP          *DatacenterSpecGCP          `json:"gcp,omitempty"`
	Fake         *DatacenterSpecFake         `json:"fake,omitempty"`
	Kubevirt     *DatacenterSpecKubevirt     `json:"kubevirt,omitempty"`

	RequiredEmailDomain string `json:"requiredEmailDomain,omitempty"`
}

// ImageList defines a map of operating system and the image to use
type ImageList map[providerconfig.OperatingSystem]string

// DatacenterSpecHetzner describes a Hetzner cloud datacenter
type DatacenterSpecHetzner struct {
	Datacenter string `json:"datacenter"`
	Location   string `json:"location"`
}

// DatacenterSpecDigitalocean describes a DigitalOcean datacenter
type DatacenterSpecDigitalocean struct {
	Region string `json:"region"`
}

// DatacenterSpecOpenstack describes an Openstack datacenter
type DatacenterSpecOpenstack struct {
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

// DatacenterSpecAzure describes an Azure cloud datacenter
type DatacenterSpecAzure struct {
	Location string `json:"location"`
}

// DatacenterSpecVSphere describes a vSphere datacenter
type DatacenterSpecVSphere struct {
	Endpoint      string `json:"endpoint"`
	AllowInsecure bool   `json:"allow_insecure"`

	Datastore  string `json:"datastore"`
	Datacenter string `json:"datacenter"`
	Cluster    string `json:"cluster"`
	// Optional root path for cluster specific VM folders.
	// Each cluster gets a own folder below the root folder.
	// Must be the FQDN (example: /datacenter-1/vm/all-kubermatic-vms-in-here )
	// Defaults to the root VM folder: /datacenter-1/vm
	RootPath  string    `json:"root_path"`
	Templates ImageList `json:"templates"`

	// Infra management user is an optional user that will be used only
	// for everything except the cloud provider functionality which will
	// still use the credentials passed in via the frontend/api
	InfraManagementUser *VSphereCredentials `json:"infra_management_user,omitempty"`
}

// DatacenterSpecAWS describes an AWS datacenter
type DatacenterSpecAWS struct {
	Region string    `json:"region"`
	Images ImageList `json:"images"`
}

// DatacenterSpecBringYourOwn describes a datacenter our of bring your own nodes
type DatacenterSpecBringYourOwn struct {
}

// DatacenterSpecPacket describes a Packet datacenter
type DatacenterSpecPacket struct {
	Facilities []string `json:"facilities"`
}

// DatacenterSpecGCP describes a GCP datacenter
type DatacenterSpecGCP struct {
	Region       string   `json:"region"`
	ZoneSuffixes []string `json:"zone_suffixes"`
	Regional     bool     `json:"regional,omitempty"`
}

// DatacenterSpecFake describes a fake datacenter
type DatacenterSpecFake struct {
	FakeProperty string `json:"fake_property,omitempty"`
}

// DatacenterSpecKubevirt describes a kubevirt datacenter.
type DatacenterSpecKubevirt struct {
}

// ProxySettings allow configuring a HTTP proxy for the controlplanes
// and nodes
type ProxySettings struct {
	// If set, this proxy will be configured for both http and https.
	HTTPProxy *string `json:"http_proxy,omitempty"`
	// If set this will be set as NO_PROXY on the node
	NoProxy *string `json:"no_proxy,omitempty"`
}

// NodeSettings are node specific flags which can be configured on datacenter level
type NodeSettings struct {
	// ProxySettings for the Nodes in this datacenter. Defaults to the HTTPProxy setting
	// on Seed level.
	ProxySettings `json:",inline"`
	// If set, this image registry will be configured as insecure on the container runtime.
	InsecureRegistries []string `json:"insecure_registries,omitempty"`
	// Translates to --pod-infra-container-image on the kubelet. If not set, the kubelet will default it
	PauseImage string `json:"pause_image,omitempty"`
	// The hyperkube image to use. Currently only Container Linux uses it
	HyperkubeImage string `json:"hyperkube_image,omitempty"`
}
