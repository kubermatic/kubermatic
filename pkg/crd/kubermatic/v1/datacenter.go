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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
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
			if dc.Node == nil {
				dc.Node = &NodeSettings{}
			}
			s.Spec.ProxySettings.Merge(&dc.Node.ProxySettings)
			s.Spec.Datacenters[key] = dc
		}
	}
}

// The spec for a seed data
type SeedSpec struct {
	// Optional: Country of the seed as ISO-3166 two-letter code, e.g. DE or UK.
	// For informational purposes in the Kubermatic dashboard only.
	Country string `json:"country,omitempty"`
	// Optional: Detailed location of the cluster, like "Hamburg" or "Datacenter 7".
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
	// Optional: This can be used to override the DNS name used for this seed.
	// By default the seed name is used.
	SeedDNSOverwrite string `json:"seed_dns_overwrite,omitempty"`
	// NodeportProxy can be used to configure the NodePort proxy service that is
	// responsible for making user-cluster control planes accessible from the outside.
	NodeportProxy NodeportProxyConfig `json:"nodeport_proxy,omitempty"`
	// Optional: ProxySettings can be used to configure HTTP proxy settings on the
	// worker nodes in user clusters. However, proxy settings on nodes take precedence.
	ProxySettings *ProxySettings `json:"proxy_settings,omitempty"`
	// Optional: ExposeStrategy explicitly sets the expose strategy for this seed cluster, if not set, the default provided by the master is used.
	ExposeStrategy corev1.ServiceType `json:"expose_strategy,omitempty"`
}

type NodeportProxyConfig struct {
	// Disable will prevent the Kubermatic Operator from creating a nodeport-proxy
	// setup on the seed cluster. This should only be used if a suitable replacement
	// is installed (like the nodeport-proxy Helm chart).
	Disable bool `json:"disable,omitempty"`
	// Annotations are used to further tweak the LoadBalancer integration with the
	// cloud provider where the seed cluster is running.
	Annotations map[string]string `json:"annotations,omitempty"`
	// Envoy configures the Envoy application itself.
	Envoy NodeportProxyComponent `json:"envoy,omitempty"`
	// EnvoyManager configures the Kubermatic-internal Envoy manager.
	EnvoyManager NodeportProxyComponent `json:"envoy_manager,omitempty"`
	// Updater configures the component responsible for updating the LoadBalancer
	// service.
	Updater NodeportProxyComponent `json:"updater,omitempty"`
}

type NodeportProxyComponent struct {
	// DockerRepository is the repository containing the component's image.
	DockerRepository string `json:"docker_repository,omitempty"`
	// Resources describes the requested and maximum allowed CPU/memory usage.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type Datacenter struct {
	// Optional: Country of the seed as ISO-3166 two-letter code, e.g. DE or UK.
	// For informational purposes in the Kubermatic dashboard only.
	Country string `json:"country,omitempty"`
	// Optional: Detailed location of the cluster, like "Hamburg" or "Datacenter 7".
	// For informational purposes in the Kubermatic dashboard only.
	Location string `json:"location,omitempty"`
	// Node holds node-specific settings, like e.g. HTTP proxy, Docker
	// registries and the like. Proxy settings are inherited from the seed if
	// not specified here.
	Node *NodeSettings `json:"node,omitempty"`
	// Spec describes the cloud provider settings used to manage resources
	// in this datacenter. Exactly one cloud provider must be defined.
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
	Alibaba      *DatacenterSpecAlibaba      `json:"alibaba,omitempty"`

	//nolint:staticcheck
	//lint:ignore SA5008 omitgenyaml is used by the example-yaml-generator
	Fake *DatacenterSpecFake `json:"fake,omitempty,omitgenyaml"`

	// Optional: When defined, only users with an e-mail address on the
	// given domains can make use of this datacenter. You can define multiple
	// domains, e.g. "example.com", one of which must match the email domain
	// exactly (i.e. "example.com" will not match "user@test.example.com").
	// RequiredEmailDomain is deprecated. Automatically migrated to the RequiredEmailDomains field.
	RequiredEmailDomain  string   `json:"requiredEmailDomain,omitempty"`
	RequiredEmailDomains []string `json:"requiredEmailDomains,omitempty"`

	// EnforceAuditLogging enforces audit logging on every cluster within the DC,
	// ignoring cluster-specific settings.
	EnforceAuditLogging bool `json:"enforceAuditLogging,omitempty"`

	// EnforcePodSecurityPolicy enforces pod security policy plugin on every clusters within the DC,
	// ignoring cluster-specific settings
	EnforcePodSecurityPolicy bool `json:"enforcePodSecurityPolicy,omitempty"`
}

// ImageList defines a map of operating system and the image to use
type ImageList map[providerconfig.OperatingSystem]string

// DatacenterSpecHetzner describes a Hetzner cloud datacenter
type DatacenterSpecHetzner struct {
	// Datacenter location, e.g. "nbg1-dc3". A list of existing datacenters can be found
	// at https://wiki.hetzner.de/index.php/Rechenzentren_und_Anbindung/en
	Datacenter string `json:"datacenter"`
	// Optional: Detailed location of the datacenter, like "Hamburg" or "Datacenter 7".
	// For informational purposes only.
	Location string `json:"location"`
}

// DatacenterSpecDigitalocean describes a DigitalOcean datacenter
type DatacenterSpecDigitalocean struct {
	// Datacenter location, e.g. "ams3". A list of existing datacenters can be found
	// at https://www.digitalocean.com/docs/platform/availability-matrix/
	Region string `json:"region"`
}

// DatacenterSpecOpenstack describes an OpenStack datacenter
type DatacenterSpecOpenstack struct {
	AuthURL          string `json:"auth_url"`
	AvailabilityZone string `json:"availability_zone"`
	Region           string `json:"region"`
	// Optional
	IgnoreVolumeAZ bool `json:"ignore_volume_az"`
	// Optional
	EnforceFloatingIP bool `json:"enforce_floating_ip"`
	// Used for automatic network creation
	DNSServers []string `json:"dns_servers"`
	// Images to use for each supported operating system.
	Images ImageList `json:"images"`
	// Optional: Gets mapped to the "manage-security-groups" setting in the cloud config.
	// See https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#load-balancer
	// This setting defaults to true.
	ManageSecurityGroups *bool `json:"manage_security_groups"`
	// Optional: Gets mapped to the "trust-device-path" setting in the cloud config.
	// See https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/#block-storage
	// This setting defaults to false.
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
	// The default Datastore to be used for provisioning volumes using storage
	// classes/dynamic provisioning and for storing virtual machine files in
	// case no `Datastore` or `DatastoreCluster` is provided at Cluster level.
	DefaultDatastore string `json:"datastore"`
	// The name of the datacenter to use.
	Datacenter string `json:"datacenter"`
	// Optional: The name of the vSphere cluster to use.
	// Cluster is deprecated and may be removed in future releases as it is
	// currently ignored.
	// The cluster hosting the VMs will be the same VM used as a template is
	// located.
	Cluster string `json:"cluster"`
	// Optional: The root path for cluster specific VM folders. Each cluster gets its own
	// folder below the root folder. Must be the FQDN (for example
	// "/datacenter-1/vm/all-kubermatic-vms-in-here") and defaults to the root VM
	// folder: "/datacenter-1/vm"
	RootPath string `json:"root_path"`
	// A list of VM templates to use for a given operating system. You must
	// define at least one template.
	// See: https://github.com/kubermatic/machine-controller/blob/master/docs/vsphere.md#template-vms-preparation
	Templates ImageList `json:"templates"`

	// Optional: Infra management user is the user that will be used for everything
	// except the cloud provider functionality, which will still use the credentials
	// passed in via the Kubermatic dashboard/API.
	InfraManagementUser *VSphereCredentials `json:"infra_management_user,omitempty"`
}

// DatacenterSpecAWS describes an AWS datacenter
type DatacenterSpecAWS struct {
	// The AWS region to use, e.g. "us-east-1". For a list of available regions, see
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html
	Region string `json:"region"`

	// List of AMIs to use for a given operating system.
	// This gets defaulted by querying for the latest AMI for the given distribution
	// when machines are created, so under normal circumstances it is not necessary
	// to define the AMIs statically.
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

	// Optional: Regional clusters spread their resources across multiple availability zones.
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

// DatacenterSpecAlibaba describes a alibaba datacenter.
type DatacenterSpecAlibaba struct {
	// Region to use, for a full list of regions see
	// https://www.alibabacloud.com/help/doc-detail/40654.htm
	Region string `json:"region"`
}

type ProxyValue string

func NewProxyValue(value string) *ProxyValue {
	val := ProxyValue(value)
	return &val
}

func (p *ProxyValue) Empty() bool {
	return p == nil || *p == ""
}

func (p *ProxyValue) String() string {
	if p.Empty() {
		return ""
	}

	return string(*p)
}

// ProxySettings allow configuring a HTTP proxy for the controlplanes
// and nodes
type ProxySettings struct {
	// Optional: If set, this proxy will be configured for both HTTP and HTTPS.
	HTTPProxy *ProxyValue `json:"http_proxy,omitempty"`
	// Optional: If set this will be set as NO_PROXY environment variable on the node;
	// The value must be a comma-separated list of domains for which no proxy
	// should be used, e.g. "*.example.com,internal.dev".
	// Note that the in-cluster apiserver URL will be automatically prepended
	// to this value.
	NoProxy *ProxyValue `json:"no_proxy,omitempty"`
}

// Empty returns true if p or all of its children are nil or empty strings.
func (p *ProxySettings) Empty() bool {
	return p == nil || (p.HTTPProxy.Empty() && p.NoProxy.Empty())
}

// Merge applies the settings from p into dst if the corresponding setting
// in dst is nil or an empty string.
func (p *ProxySettings) Merge(dst *ProxySettings) {
	if dst.HTTPProxy.Empty() {
		dst.HTTPProxy = p.HTTPProxy
	}
	if dst.NoProxy.Empty() {
		dst.NoProxy = p.NoProxy
	}
}

// NodeSettings are node specific flags which can be configured on datacenter level
type NodeSettings struct {
	// Optional: Proxy settings for the Nodes in this datacenter.
	// Defaults to the Proxy settings of the seed.
	ProxySettings `json:",inline"`
	// Optional: These image registries will be configured as insecure
	// on the container runtime.
	InsecureRegistries []string `json:"insecure_registries,omitempty"`
	// Optional: Translates to --pod-infra-container-image on the kubelet.
	// If not set, the kubelet will default it.
	PauseImage string `json:"pause_image,omitempty"`
	// Optional: The hyperkube image to use. Currently only Container Linux
	// makes use of this option.
	HyperkubeImage string `json:"hyperkube_image,omitempty"`
}
