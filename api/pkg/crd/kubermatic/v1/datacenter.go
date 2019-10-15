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
	// apply seed-level proxy settings to all datacenters, if the datacenters have no
	// settings on their own
	if !s.Spec.ProxySettings.Empty() {
		for key, dc := range s.Spec.Datacenters {
			s.Spec.ProxySettings.Merge(&dc.Node.ProxySettings)
			s.Spec.Datacenters[key] = dc
		}
	}
}

// The spec for a seed data
type SeedSpec struct {
	// Country of the seed as ISO-3166 two-letter code, e.g. DE or UK.
	// For informational purposes in the Kubermatic dashboard only.
	Country string `json:"country,omitempty"`
	// Detailed location of the cluster, like "Hamburg" or "Datacenter 7".
	// For informational purposes in the Kubermatic dashboard only.
	Location string `json:"location,omitempty"`
	// A reference to the Kubeconfig of this cluster. The Kubeconfig must
	// have cluster-admin privileges. This field is mandatory for every
	// seed, even if there are no datacenters defined yet.
	Kubeconfig corev1.ObjectReference `json:"kubeconfig"`
	// Datacenters contains a map of the possible datacenters (DCs) in this seed.
	// Each DC must have a globally unique identifier (i.e. names must be unique
	// across all seeds).
	Datacenters map[string]Datacenter `json:"datacenters,omitempty"`
	// SeedDNSOverwrite can be used to override the DNS name used for this seed.
	// This field is optional and defaults to the seed name.
	SeedDNSOverwrite *string `json:"seed_dns_overwrite,omitempty"`
	// ProxySettings can optionally be used to configure HTTP proxy settings on the
	// worker nodes in user clusters. However, proxy settings on nodes take
	// precedence.
	ProxySettings *ProxySettings `json:"proxy_settings,omitempty"`
}

type Datacenter struct {
	// Country of the seed as ISO-3166 two-letter code, e.g. DE or UK.
	// For informational purposes in the Kubermatic dashboard only.
	Country string `json:"country,omitempty"`
	// Detailed location of the cluster, like "Hamburg" or "Datacenter 7".
	// For informational purposes in the Kubermatic dashboard only.
	Location string `json:"location,omitempty"`
	// Node holds node-specific settings, like e.g. HTTP proxy and Docker
	// registries and the like. Proxy settings are inherited from the seed if
	// not specified here.
	Node NodeSettings `json:"node"`
	// Spec describes the cloud provider settings used to manage resources
	// in this datacenter. Exactly one cloud provider must be defined and it
	// must not be the internal "fake" provider.
	Spec DatacenterSpec `json:"spec"`
}

// DatacenterSpec mutually points to provider datacenter spec
type DatacenterSpec struct {
	Digitalocean *DatacenterSpecDigitalocean `json:"digitalocean,omitempty"`
	// BringYourOwn contains settings for clusters using manually created
	// nodes via kubeadm.
	BringYourOwn *DatacenterSpecBringYourOwn `json:"bringyourown,omitempty"`
	AWS          *DatacenterSpecAWS          `json:"aws,omitempty"`
	Azure        *DatacenterSpecAzure        `json:"azure,omitempty"`
	Openstack    *DatacenterSpecOpenstack    `json:"openstack,omitempty"`
	Packet       *DatacenterSpecPacket       `json:"packet,omitempty"`
	Hetzner      *DatacenterSpecHetzner      `json:"hetzner,omitempty"`
	VSphere      *DatacenterSpecVSphere      `json:"vsphere,omitempty"`
	GCP          *DatacenterSpecGCP          `json:"gcp,omitempty"`
	Kubevirt     *DatacenterSpecKubevirt     `json:"kubevirt,omitempty"`

	// This is only for internal use and must not be set externally.
	Fake *DatacenterSpecFake `json:"fake,omitempty,omitgenyaml"` // omitgenyaml is used by the example-yaml-generator

	// By specifying a required email domain, only users with an e-mail
	// address on the given domain can make use of this datacenter. You
	// can define exactly one domain, e.g. "example.com".
	RequiredEmailDomain string `json:"requiredEmailDomain,omitempty"`
}

// ImageList defines a map of operating system and the image to use
type ImageList map[providerconfig.OperatingSystem]string

// DatacenterSpecHetzner describes a Hetzner cloud datacenter
type DatacenterSpecHetzner struct {
	// Datacenter location, e.g. "nbg1-dc3". A list of existing datacenters can be found
	// at https://wiki.hetzner.de/index.php/Rechenzentren_und_Anbindung/en
	Datacenter string `json:"datacenter"`
	// Detailed location of the datacenter, like "Hamburg" or "Datacenter 7".
	// For informational purposes only.
	Location string `json:"location"`
}

// DatacenterSpecDigitalocean describes a DigitalOcean datacenter
type DatacenterSpecDigitalocean struct {
	// Datacenter location, e.g. "ams3". A list of existing datacenters can be found
	// at https://www.digitalocean.com/docs/platform/availability-matrix/
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
	// Region to use, for example "westeurope". A list of available regions can be
	// found at https://azure.microsoft.com/en-us/global-infrastructure/locations/
	Location string `json:"location"`
}

// DatacenterSpecVSphere describes a vSphere datacenter
type DatacenterSpecVSphere struct {
	// Endpoint URL to use, including protocol, for example "https://vcenter.example.com".
	Endpoint string `json:"endpoint"`
	// If set to true, disables the TLS certificate check against the endpoint.
	AllowInsecure bool `json:"allow_insecure"`
	// The name of the datastore to use.
	Datastore string `json:"datastore"`
	// The name of the datacenter to use.
	Datacenter string `json:"datacenter"`
	// The name of the cluster to use.
	Cluster string `json:"cluster"`
	// Optional root path for cluster specific VM folders. Each cluster gets its own
	// folder below the root folder. Must be the FQDN (for example
	// "/datacenter-1/vm/all-kubermatic-vms-in-here") and defaults to the root VM
	// folder: "/datacenter-1/vm"
	RootPath string `json:"root_path"`
	// A list con templates to use for a given operating system. You must define at
	// least one template.
	Templates ImageList `json:"templates"`

	// Infra management user is an optional user that will be used for everything
	// except the cloud provider functionality which will still use the credentials
	// passed in via the Kubermatic dashboard/API.
	InfraManagementUser *VSphereCredentials `json:"infra_management_user,omitempty"`
}

// DatacenterSpecAWS describes an AWS datacenter
type DatacenterSpecAWS struct {
	// The AWS region to use, e.g. "us-east-1". For a list of available regions, see
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html
	Region string `json:"region"`

	// List of AMIs to use for a given operating system.
	// TODO: Explain how the default for this is assembled.
	Images ImageList `json:"images"`
}

// DatacenterSpecBringYourOwn describes a datacenter our of bring your own nodes
type DatacenterSpecBringYourOwn struct {
}

// DatacenterSpecPacket describes a Packet datacenter
type DatacenterSpecPacket struct {
	// The list of enabled facilities, for example "ams1", for a full list of available
	// facilities see https://support.packet.com/kb/articles/data-centers
	Facilities []string `json:"facilities"`
}

// DatacenterSpecGCP describes a GCP datacenter
type DatacenterSpecGCP struct {
	// Region to use, for example "europe-west3", for a full list of regions see
	// https://cloud.google.com/compute/docs/regions-zones/
	Region string `json:"region"`
	// List of enabled zones, for example [a, c]. See the link above for the available
	// zones in your chosen region.
	ZoneSuffixes []string `json:"zone_suffixes"`

	// Regional clusters spread their resources across multiple availability zones.
	// Refer to the official documentation for more details on this:
	// https://cloud.google.com/kubernetes-engine/docs/concepts/regional-clusters
	Regional bool `json:"regional,omitempty"`
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
	// If set, this proxy will be configured for both HTTP and HTTPS.
	HTTPProxy *string `json:"http_proxy,omitempty"`
	// If set this will be set as NO_PROXY environment variable on the node;
	// The value must be a comma-separated list of domains for which no proxy
	// should be used, e.g. "*.example.com,internal.dev"
	NoProxy *string `json:"no_proxy,omitempty"`
}

// Empty returns true if p or any of its children are nil or empty strings.
func (p *ProxySettings) Empty() bool {
	return p == nil || (emptyStr(p.HTTPProxy) && emptyStr(p.NoProxy))
}

// Merge applies the settings from p into dst if the corresponding setting
// in dst is nil or an empty string.
func (p *ProxySettings) Merge(dst *ProxySettings) {
	if emptyStr(dst.HTTPProxy) {
		dst.HTTPProxy = p.HTTPProxy
	}
	if emptyStr(dst.NoProxy) {
		dst.NoProxy = p.NoProxy
	}
}

func emptyStr(s *string) bool {
	return s == nil || *s == ""
}

// NodeSettings are node specific flags which can be configured on datacenter level
type NodeSettings struct {
	// ProxySettings for the Nodes in this datacenter. Defaults to the HTTPProxy setting
	// on Seed level.
	ProxySettings `json:",inline"`
	// If set, these image registries will be configured as insecure on the container runtime.
	InsecureRegistries []string `json:"insecure_registries,omitempty"`
	// Translates to --pod-infra-container-image on the kubelet. If not set, the kubelet will default it
	PauseImage string `json:"pause_image,omitempty"`
	// The hyperkube image to use. Currently only Container Linux uses it
	HyperkubeImage string `json:"hyperkube_image,omitempty"`
}
