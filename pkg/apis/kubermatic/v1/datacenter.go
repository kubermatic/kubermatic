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
	"strings"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=digitalocean;hetzner;azure;vsphere;aws;openstack;packet;gcp;kubevirt;nutanix;alibaba;anexia;fake;vmwareclouddirector

type ProviderType string

// +kubebuilder:validation:Pattern:=`^((\d{1,3}\.){3}\d{1,3}\/([0-9]|[1-2][0-9]|3[0-2]))$`
type CIDR string

const (
	// Constants defining known cloud providers.
	FakeCloudProvider                ProviderType = "fake"
	AKSCloudProvider                 ProviderType = "aks"
	AlibabaCloudProvider             ProviderType = "alibaba"
	AnexiaCloudProvider              ProviderType = "anexia"
	AWSCloudProvider                 ProviderType = "aws"
	AzureCloudProvider               ProviderType = "azure"
	BringYourOwnCloudProvider        ProviderType = "bringyourown"
	DigitaloceanCloudProvider        ProviderType = "digitalocean"
	EKSCloudProvider                 ProviderType = "eks"
	GCPCloudProvider                 ProviderType = "gcp"
	GKECloudProvider                 ProviderType = "gke"
	HetznerCloudProvider             ProviderType = "hetzner"
	KubevirtCloudProvider            ProviderType = "kubevirt"
	NutanixCloudProvider             ProviderType = "nutanix"
	OpenstackCloudProvider           ProviderType = "openstack"
	PacketCloudProvider              ProviderType = "packet"
	VMwareCloudDirectorCloudProvider ProviderType = "vmwareclouddirector"
	VSphereCloudProvider             ProviderType = "vsphere"

	DefaultSSHPort     = 22
	DefaultKubeletPort = 10250

	DefaultKubeconfigFieldPath = "kubeconfig"
)

var (
	SupportedProviders = []ProviderType{
		AKSCloudProvider,
		AlibabaCloudProvider,
		AnexiaCloudProvider,
		AWSCloudProvider,
		AzureCloudProvider,
		BringYourOwnCloudProvider,
		DigitaloceanCloudProvider,
		EKSCloudProvider,
		FakeCloudProvider,
		GCPCloudProvider,
		GKECloudProvider,
		HetznerCloudProvider,
		KubevirtCloudProvider,
		NutanixCloudProvider,
		OpenstackCloudProvider,
		PacketCloudProvider,
		VMwareCloudDirectorCloudProvider,
		VSphereCloudProvider,
	}
)

func IsProviderSupported(name string) bool {
	for _, provider := range SupportedProviders {
		if strings.EqualFold(name, string(provider)) {
			return true
		}
	}

	return false
}

// +kubebuilder:validation:Enum="";Healthy;Unhealthy;Invalid;Terminating;Paused

type SeedPhase string

// These are the valid phases of a seed.
const (
	// SeedHealthyPhase means the seed is reachable and was successfully reconciled.
	SeedHealthyPhase SeedPhase = "Healthy"

	// SeedUnhealthyPhase means the KKP resources on the seed cluster could not be
	// successfully reconciled.
	SeedUnhealthyPhase SeedPhase = "Unhealthy"

	// SeedInvalidPhase means the seed kubeconfig is defunct.
	SeedInvalidPhase SeedPhase = "Invalid"

	// SeedTerminatingPhase means the seed is currently being deleted.
	SeedTerminatingPhase SeedPhase = "Terminating"

	// SeedPausedPhase means the seed is not being reconciled because the SkipReconciling
	// annotation is set.
	SeedPausedPhase SeedPhase = "Paused"
)

// +kubebuilder:validation:Enum="";KubeconfigValid;ResourcesReconciled

// SeedConditionType is used to indicate the type of a seed condition. For all condition
// types, the `true` value must indicate success. All condition types must be registered
// within the `AllSeedConditionTypes` variable.
type SeedConditionType string

const (
	// SeedConditionKubeconfigValid indicates that the configured kubeconfig for the seed is valid.
	// The seed-sync controller manages this condition.
	SeedConditionKubeconfigValid SeedConditionType = "KubeconfigValid"
	// SeedConditionResourcesReconciled indicates that the KKP operator has finished setting up the
	// resources inside the seed cluster.
	SeedConditionResourcesReconciled SeedConditionType = "ResourcesReconciled"
)

var AllSeedConditionTypes = []SeedConditionType{
	SeedConditionResourcesReconciled,
}

type SeedCondition struct {
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time we got an update on a given condition.
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// SeedDatacenterList is the type representing a SeedDatacenterList.
type SeedList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// List of seeds
	Items []Seed `json:"items"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:JSONPath=".status.clusters",name="Clusters",type="integer"
// +kubebuilder:printcolumn:JSONPath=".spec.location",name="Location",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.versions.kubermatic",name="KKP Version",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.versions.cluster",name="Cluster Version",type="string"
// +kubebuilder:printcolumn:JSONPath=".status.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// Seed is the type representing a Seed cluster. Seed clusters host the the control planes
// for KKP user clusters.
type Seed struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SeedSpec `json:"spec"`
	//nolint:staticcheck
	//lint:ignore SA5008 omitgenyaml is used by the example-yaml-generator
	Status SeedStatus `json:"status,omitempty,omitgenyaml"`
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

// SeedStatus contains runtime information regarding the seed.
type SeedStatus struct {
	// Phase contains a human readable text to indicate the seed cluster status. No logic should be tied
	// to this field, as its content can change in between KKP releases.
	Phase SeedPhase `json:"phase,omitempty"`

	// +kubebuilder:default=0
	// +kubebuilder:validation:Minimum:=0

	// Clusters is the total number of user clusters that exist on this seed.
	Clusters int `json:"clusters"`

	// Versions contains information regarding versions of components in the cluster and the cluster
	// itself.
	// +optional
	Versions SeedVersionsStatus `json:"versions,omitempty"`

	// Conditions contains conditions the seed is in, its primary use case is status signaling
	// between controllers or between controllers and the API.
	// +optional
	Conditions map[SeedConditionType]SeedCondition `json:"conditions,omitempty"`
}

// SeedVersionsStatus contains information regarding versions of components in the cluster
// and the cluster itself.
type SeedVersionsStatus struct {
	// Kubermatic is the version of the currently deployed KKP components. Note that a permanent
	// version skew between master and seed is not supported and KKP setups should never run for
	// longer times with a skew between the clusters.
	Kubermatic string `json:"kubermatic,omitempty"`
	// Cluster is the Kubernetes version of the cluster's control plane.
	Cluster string `json:"cluster,omitempty"`
}

// HasConditionValue returns true if the seed status has the given condition with the given status.
func (ss *SeedStatus) HasConditionValue(conditionType SeedConditionType, conditionStatus corev1.ConditionStatus) bool {
	condition, exists := ss.Conditions[conditionType]
	if !exists {
		return false
	}

	return condition.Status == conditionStatus
}

// The spec for a seed cluster.
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
	SeedDNSOverwrite string `json:"seedDNSOverwrite,omitempty"`
	// NodeportProxy can be used to configure the NodePort proxy service that is
	// responsible for making user-cluster control planes accessible from the outside.
	NodeportProxy NodeportProxyConfig `json:"nodeportProxy,omitempty"`
	// Optional: ProxySettings can be used to configure HTTP proxy settings on the
	// worker nodes in user clusters. However, proxy settings on nodes take precedence.
	ProxySettings *ProxySettings `json:"proxySettings,omitempty"`
	// Optional: ExposeStrategy explicitly sets the expose strategy for this seed cluster, if not set, the default provided by the master is used.
	ExposeStrategy ExposeStrategy `json:"exposeStrategy,omitempty"`
	// Optional: MLA allows configuring seed level MLA (Monitoring, Logging & Alerting) stack settings.
	MLA *SeedMLASettings `json:"mla,omitempty"`
	// DefaultComponentSettings are default values to set for newly created clusters.
	// Deprecated: Use DefaultClusterTemplate instead.
	DefaultComponentSettings ComponentSettings `json:"defaultComponentSettings,omitempty"`
	// DefaultClusterTemplate is the name of a cluster template of scope "seed" that is used
	// to default all new created clusters
	DefaultClusterTemplate string `json:"defaultClusterTemplate,omitempty"`
	// Metering configures the metering tool on user clusters across the seed.
	Metering *MeteringConfiguration `json:"metering,omitempty"`
	// EtcdBackupRestore holds the configuration of the automatic etcd backup restores for the Seed;
	// if this is set, the new backup/restore controllers are enabled for this Seed.
	EtcdBackupRestore *EtcdBackupRestore `json:"etcdBackupRestore,omitempty"`
}

// EtcdBackupRestore holds the configuration of the automatic backup and restores.
type EtcdBackupRestore struct {
	// Destinations stores all the possible destinations where the backups for the Seed can be stored. If not empty,
	// it enables automatic backup and restore for the seed.
	Destinations map[string]*BackupDestination `json:"destinations,omitempty"`

	// +kubebuilder:validation:Pattern:=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Type=string

	// DefaultDestination marks the default destination that will be used for the default etcd backup config which is
	// created for every user cluster. Has to correspond to a destination in Destinations.
	// If removed, it removes the related default etcd backup configs.
	DefaultDestination string `json:"defaultDestination,omitempty"`
}

// BackupDestination defines the bucket name and endpoint as a backup destination, and holds reference to the credentials secret.
type BackupDestination struct {
	// Endpoint is the API endpoint to use for backup and restore.
	Endpoint string `json:"endpoint"`
	// BucketName is the bucket name to use for backup and restore.
	BucketName string `json:"bucketName"`
	// Credentials hold the ref to the secret with backup credentials
	Credentials *corev1.SecretReference `json:"credentials,omitempty"`
}

type NodeportProxyConfig struct {
	// Disable will prevent the Kubermatic Operator from creating a nodeport-proxy
	// setup on the seed cluster. This should only be used if a suitable replacement
	// is installed (like the nodeport-proxy Helm chart).
	Disable bool `json:"disable,omitempty"`
	// Annotations are used to further tweak the LoadBalancer integration with the
	// cloud provider where the seed cluster is running.
	// Deprecated: Use .envoy.loadBalancerService.annotations instead.
	Annotations map[string]string `json:"annotations,omitempty"`
	// Envoy configures the Envoy application itself.
	Envoy NodePortProxyComponentEnvoy `json:"envoy,omitempty"`
	// EnvoyManager configures the Kubermatic-internal Envoy manager.
	EnvoyManager NodeportProxyComponent `json:"envoyManager,omitempty"`
	// Updater configures the component responsible for updating the LoadBalancer
	// service.
	Updater NodeportProxyComponent `json:"updater,omitempty"`
}

type EnvoyLoadBalancerService struct {
	// Annotations are used to further tweak the LoadBalancer integration with the
	// cloud provider.
	Annotations map[string]string `json:"annotations,omitempty"`
	// SourceRanges will restrict loadbalancer service to IP ranges specified using CIDR notation like 172.25.0.0/16.
	// This field will be ignored if the cloud-provider does not support the feature.
	// More info: https://kubernetes.io/docs/tasks/access-application-cluster/create-external-load-balancer/
	SourceRanges []CIDR `json:"sourceRanges,omitempty"`
}
type NodePortProxyComponentEnvoy struct {
	NodeportProxyComponent `json:",inline"`
	LoadBalancerService    EnvoyLoadBalancerService `json:"loadBalancerService,omitempty"`
}

type NodeportProxyComponent struct {
	// DockerRepository is the repository containing the component's image.
	DockerRepository string `json:"dockerRepository,omitempty"`
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

// DatacenterSpec mutually points to provider datacenter spec.
type DatacenterSpec struct {
	Digitalocean *DatacenterSpecDigitalocean `json:"digitalocean,omitempty"`
	// BringYourOwn contains settings for clusters using manually created
	// nodes via kubeadm.
	BringYourOwn        *DatacenterSpecBringYourOwn        `json:"bringyourown,omitempty"`
	AWS                 *DatacenterSpecAWS                 `json:"aws,omitempty"`
	Azure               *DatacenterSpecAzure               `json:"azure,omitempty"`
	Openstack           *DatacenterSpecOpenstack           `json:"openstack,omitempty"`
	Packet              *DatacenterSpecPacket              `json:"packet,omitempty"`
	Hetzner             *DatacenterSpecHetzner             `json:"hetzner,omitempty"`
	VSphere             *DatacenterSpecVSphere             `json:"vsphere,omitempty"`
	VMwareCloudDirector *DatacenterSpecVMwareCloudDirector `json:"vmwareclouddirector,omitempty"`
	GCP                 *DatacenterSpecGCP                 `json:"gcp,omitempty"`
	Kubevirt            *DatacenterSpecKubevirt            `json:"kubevirt,omitempty"`
	Alibaba             *DatacenterSpecAlibaba             `json:"alibaba,omitempty"`
	Anexia              *DatacenterSpecAnexia              `json:"anexia,omitempty"`
	// Nutanix is experimental and unsupported
	Nutanix *DatacenterSpecNutanix `json:"nutanix,omitempty"`

	//nolint:staticcheck
	//lint:ignore SA5008 omitgenyaml is used by the example-yaml-generator
	Fake *DatacenterSpecFake `json:"fake,omitempty,omitgenyaml"`

	// Optional: When defined, only users with an e-mail address on the
	// given domains can make use of this datacenter. You can define multiple
	// domains, e.g. "example.com", one of which must match the email domain
	// exactly (i.e. "example.com" will not match "user@test.example.com").
	RequiredEmails []string `json:"requiredEmails,omitempty"`

	// EnforceAuditLogging enforces audit logging on every cluster within the DC,
	// ignoring cluster-specific settings.
	EnforceAuditLogging bool `json:"enforceAuditLogging,omitempty"`

	// EnforcePodSecurityPolicy enforces pod security policy plugin on every clusters within the DC,
	// ignoring cluster-specific settings
	EnforcePodSecurityPolicy bool `json:"enforcePodSecurityPolicy,omitempty"`

	// ProviderReconciliationInterval is the time that must have passed since a
	// Cluster's status.lastProviderReconciliation to make the cliuster controller
	// perform an in-depth provider reconciliation, where for example missing security
	// groups will be reconciled.
	// Setting this too low can cause rate limits by the cloud provider, setting this
	// too high means that *if* a resource at a cloud provider is removed/changed outside
	// of KKP, it will take this long to fix it.
	ProviderReconciliationInterval *metav1.Duration `json:"providerReconciliationInterval,omitempty"`
}

var (
	// knownIPv6CloudProviders configures which providers have IPv6 and if it's enabled for all datacenters.
	knownIPv6CloudProviders = map[ProviderType]struct {
		ipv6EnabledForAllDatacenters bool
	}{
		AWSCloudProvider: {
			ipv6EnabledForAllDatacenters: true,
		},
		AzureCloudProvider: {
			ipv6EnabledForAllDatacenters: true,
		},
		BringYourOwnCloudProvider: {
			ipv6EnabledForAllDatacenters: true,
		},
		DigitaloceanCloudProvider: {
			ipv6EnabledForAllDatacenters: true,
		},
		GCPCloudProvider: {
			ipv6EnabledForAllDatacenters: true,
		},
		HetznerCloudProvider: {
			ipv6EnabledForAllDatacenters: true,
		},
		OpenstackCloudProvider: {
			ipv6EnabledForAllDatacenters: false,
		},
		PacketCloudProvider: {
			ipv6EnabledForAllDatacenters: true,
		},
		VSphereCloudProvider: {
			ipv6EnabledForAllDatacenters: false,
		},
	}
)

func (cloudProvider ProviderType) IsIPv6KnownProvider() bool {
	_, isIPv6KnownProvider := knownIPv6CloudProviders[cloudProvider]
	return isIPv6KnownProvider
}

// IsIPv6Enabled returns true if ipv6 is enabled for the datacenter.
func (d *Datacenter) IsIPv6Enabled(cloudProvider ProviderType) bool {
	cloudProviderCfg, isIPv6KnownProvider := knownIPv6CloudProviders[cloudProvider]
	if !isIPv6KnownProvider {
		return false
	}

	if cloudProviderCfg.ipv6EnabledForAllDatacenters {
		return true
	}

	switch cloudProvider {
	case OpenstackCloudProvider:
		if d.Spec.Openstack != nil && d.Spec.Openstack.IPv6Enabled != nil && *d.Spec.Openstack.IPv6Enabled {
			return true
		}
	case VSphereCloudProvider:
		if d.Spec.VSphere != nil && d.Spec.VSphere.IPv6Enabled != nil && *d.Spec.VSphere.IPv6Enabled {
			return true
		}
	}

	return false
}

// ImageList defines a map of operating system and the image to use.
type ImageList map[providerconfig.OperatingSystem]string

// DatacenterSpecHetzner describes a Hetzner cloud datacenter.
type DatacenterSpecHetzner struct {
	// Datacenter location, e.g. "nbg1-dc3". A list of existing datacenters can be found
	// at https://docs.hetzner.com/general/others/data-centers-and-connection/
	Datacenter string `json:"datacenter"`
	// Network is the pre-existing Hetzner network in which the machines are running.
	// While machines can be in multiple networks, a single one must be chosen for the
	// HCloud CCM to work.
	Network string `json:"network"`
	// Optional: Detailed location of the datacenter, like "Hamburg" or "Datacenter 7".
	// For informational purposes only.
	Location string `json:"location,omitempty"`
}

// DatacenterSpecDigitalocean describes a DigitalOcean datacenter.
type DatacenterSpecDigitalocean struct {
	// Datacenter location, e.g. "ams3". A list of existing datacenters can be found
	// at https://www.digitalocean.com/docs/platform/availability-matrix/
	Region string `json:"region"`
}

// DatacenterSpecOpenstack describes an OpenStack datacenter.
type DatacenterSpecOpenstack struct {
	AuthURL          string `json:"authURL"`
	AvailabilityZone string `json:"availabilityZone"`
	Region           string `json:"region"`
	// Optional
	IgnoreVolumeAZ bool `json:"ignoreVolumeAZ,omitempty"` //nolint:tagliatelle
	// Optional
	EnforceFloatingIP bool `json:"enforceFloatingIP,omitempty"`
	// Used for automatic network creation
	DNSServers []string `json:"dnsServers"`
	// Images to use for each supported operating system.
	Images ImageList `json:"images"`
	// Optional: Gets mapped to the "manage-security-groups" setting in the cloud config.
	// This setting defaults to true.
	ManageSecurityGroups *bool `json:"manageSecurityGroups,omitempty"`
	// Optional: Gets mapped to the "use-octavia" setting in the cloud config.
	// use-octavia is enabled by default in CCM since v1.17.0, and disabled by
	// default with the in-tree cloud provider.
	UseOctavia *bool `json:"useOctavia,omitempty"`
	// Optional: Gets mapped to the "trust-device-path" setting in the cloud config.
	// This setting defaults to false.
	TrustDevicePath      *bool                         `json:"trustDevicePath,omitempty"`
	NodeSizeRequirements OpenstackNodeSizeRequirements `json:"nodeSizeRequirements"`
	// Optional: List of enabled flavors for the given datacenter
	EnabledFlavors []string `json:"enabledFlavors,omitempty"`
	// Optional: defines if the IPv6 is enabled for the datacenter
	IPv6Enabled *bool `json:"ipv6Enabled,omitempty"`
}

type OpenstackNodeSizeRequirements struct {
	// VCPUs is the minimum required amount of (virtual) CPUs
	MinimumVCPUs int `json:"minimumVCPUs"` //nolint:tagliatelle
	// MinimumMemory is the minimum required amount of memory, measured in MB
	MinimumMemory int `json:"minimumMemory"`
}

// DatacenterSpecAzure describes an Azure cloud datacenter.
type DatacenterSpecAzure struct {
	// Region to use, for example "westeurope". A list of available regions can be
	// found at https://azure.microsoft.com/en-us/global-infrastructure/locations/
	Location string `json:"location"`
}

// DatacenterSpecVSphere describes a vSphere datacenter.
type DatacenterSpecVSphere struct {
	// Endpoint URL to use, including protocol, for example "https://vcenter.example.com".
	Endpoint string `json:"endpoint"`
	// If set to true, disables the TLS certificate check against the endpoint.
	AllowInsecure bool `json:"allowInsecure,omitempty"`
	// The default Datastore to be used for provisioning volumes using storage
	// classes/dynamic provisioning and for storing virtual machine files in
	// case no `Datastore` or `DatastoreCluster` is provided at Cluster level.
	DefaultDatastore string `json:"datastore"`
	// The name of the datacenter to use.
	Datacenter string `json:"datacenter"`
	// The name of the vSphere cluster to use. Used for out-of-tree CSI Driver.
	Cluster string `json:"cluster"`
	// The name of the storage policy to use for the storage class created in the user cluster.
	DefaultStoragePolicy string `json:"storagePolicy,omitempty"`
	// Optional: The root path for cluster specific VM folders. Each cluster gets its own
	// folder below the root folder. Must be the FQDN (for example
	// "/datacenter-1/vm/all-kubermatic-vms-in-here") and defaults to the root VM
	// folder: "/datacenter-1/vm"
	RootPath string `json:"rootPath,omitempty"`
	// A list of VM templates to use for a given operating system. You must
	// define at least one template.
	// See: https://github.com/kubermatic/machine-controller/blob/master/docs/vsphere.md#template-vms-preparation
	Templates ImageList `json:"templates"`
	// Optional: Infra management user is the user that will be used for everything
	// except the cloud provider functionality, which will still use the credentials
	// passed in via the Kubermatic dashboard/API.
	InfraManagementUser *VSphereCredentials `json:"infraManagementUser,omitempty"`
	// Optional: defines if the IPv6 is enabled for the datacenter
	IPv6Enabled *bool `json:"ipv6Enabled,omitempty"`
	// DefaultTagCategoryID is the tag category id that will be used as default, if users don't specify it on a cluster level,
	// and they don't wish KKP to create default generated tag category, upon cluster creation.
	DefaultTagCategoryID string `json:"defaultTagCategoryID,omitempty"`
}

type DatacenterSpecVMwareCloudDirector struct {
	// Endpoint URL to use, including protocol, for example "https://vclouddirector.example.com".
	URL string `json:"url"`
	// If set to true, disables the TLS certificate check against the endpoint.
	AllowInsecure bool `json:"allowInsecure,omitempty"`
	// The default catalog which contains the VM templates.
	DefaultCatalog string `json:"catalog,omitempty"`
	// The name of the storage profile to use for disks attached to the VMs.
	DefaultStorageProfile string `json:"storageProfile,omitempty"`
	// A list of VM templates to use for a given operating system. You must
	// define at least one template.
	Templates ImageList `json:"templates"`
}

// DatacenterSpecAWS describes an AWS datacenter.
type DatacenterSpecAWS struct {
	// The AWS region to use, e.g. "us-east-1". For a list of available regions, see
	// https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/using-regions-availability-zones.html
	Region string `json:"region"`

	// List of AMIs to use for a given operating system.
	// This gets defaulted by querying for the latest AMI for the given distribution
	// when machines are created, so under normal circumstances it is not necessary
	// to define the AMIs statically.
	Images ImageList `json:"images,omitempty"`
}

// DatacenterSpecBringYourOwn describes a datacenter our of bring your own nodes.
type DatacenterSpecBringYourOwn struct {
}

// DatacenterSpecPacket describes a Packet datacenter.
type DatacenterSpecPacket struct {
	// The list of enabled facilities, for example "ams1", for a full list of available
	// facilities see https://metal.equinix.com/developers/docs/locations/facilities/
	Facilities []string `json:"facilities,omitempty"`
	// Metros are facilities that are grouped together geographically and share capacity
	// and networking features, see https://metal.equinix.com/developers/docs/locations/metros/
	Metro string `json:"metro,omitempty"`
}

// DatacenterSpecGCP describes a GCP datacenter.
type DatacenterSpecGCP struct {
	// Region to use, for example "europe-west3", for a full list of regions see
	// https://cloud.google.com/compute/docs/regions-zones/
	Region string `json:"region"`
	// List of enabled zones, for example [a, c]. See the link above for the available
	// zones in your chosen region.
	ZoneSuffixes []string `json:"zoneSuffixes"`

	// Optional: Regional clusters spread their resources across multiple availability zones.
	// Refer to the official documentation for more details on this:
	// https://cloud.google.com/kubernetes-engine/docs/concepts/regional-clusters
	Regional bool `json:"regional,omitempty"`
}

// DatacenterSpecFake describes a fake datacenter.
type DatacenterSpecFake struct {
	FakeProperty string `json:"fakeProperty,omitempty"`
}

// DatacenterSpecKubevirt describes a kubevirt datacenter.
type DatacenterSpecKubevirt struct {
	// +kubebuilder:validation:Enum=ClusterFirstWithHostNet;ClusterFirst;Default;None
	// +kubebuilder:default=ClusterFirst

	// DNSPolicy represents the dns policy for the pod. Valid values are 'ClusterFirstWithHostNet', 'ClusterFirst',
	// 'Default' or 'None'. Defaults to "ClusterFirst". DNS parameters given in DNSConfig will be merged with the
	// policy selected with DNSPolicy.
	DNSPolicy string `json:"dnsPolicy,omitempty"`

	// DNSConfig represents the DNS parameters of a pod. Parameters specified here will be merged to the generated DNS
	// configuration based on DNSPolicy.
	DNSConfig *corev1.PodDNSConfig `json:"dnsConfig,omitempty"`
}

// DatacenterSpecNutanix describes a Nutanix datacenter.
type DatacenterSpecNutanix struct {
	// Endpoint to use for accessing Nutanix Prism Central. No protocol or port should be passed,
	// for example "nutanix.example.com" or "10.0.0.1"
	Endpoint string `json:"endpoint"`
	// Optional: Port to use when connecting to the Nutanix Prism Central endpoint (defaults to 9440)
	Port *int32 `json:"port,omitempty"`

	// Optional: AllowInsecure allows to disable the TLS certificate check against the endpoint (defaults to false)
	AllowInsecure bool `json:"allowInsecure,omitempty"`
	// Images to use for each supported operating system
	Images ImageList `json:"images"`
}

// DatacenterSpecAlibaba describes a alibaba datacenter.
type DatacenterSpecAlibaba struct {
	// Region to use, for a full list of regions see
	// https://www.alibabacloud.com/help/doc-detail/40654.htm
	Region string `json:"region"`
}

// DatacenterSpecAnexia describes a anexia datacenter.
type DatacenterSpecAnexia struct {
	// LocationID the location of the region
	LocationID string `json:"locationID"`
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
// and nodes.
type ProxySettings struct {
	// Optional: If set, this proxy will be configured for both HTTP and HTTPS.
	HTTPProxy *ProxyValue `json:"httpProxy,omitempty"`
	// Optional: If set this will be set as NO_PROXY environment variable on the node;
	// The value must be a comma-separated list of domains for which no proxy
	// should be used, e.g. "*.example.com,internal.dev".
	// Note that the in-cluster apiserver URL will be automatically prepended
	// to this value.
	NoProxy *ProxyValue `json:"noProxy,omitempty"`
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

// NodeSettings are node specific flags which can be configured on datacenter level.
type NodeSettings struct {
	// Optional: Proxy settings for the Nodes in this datacenter.
	// Defaults to the Proxy settings of the seed.
	ProxySettings `json:",inline"`
	// Optional: These image registries will be configured as insecure
	// on the container runtime.
	InsecureRegistries []string `json:"insecureRegistries,omitempty"`
	// Optional: These image registries will be configured as registry mirrors
	// on the container runtime.
	RegistryMirrors []string `json:"registryMirrors,omitempty"`
	// Optional: Translates to --pod-infra-container-image on the kubelet.
	// If not set, the kubelet will default it.
	PauseImage string `json:"pauseImage,omitempty"`
	// Optional: ContainerdRegistryMirrors configure registry mirrors endpoints. Can be used multiple times to specify multiple mirrors.
	ContainerdRegistryMirrors *ContainerRuntimeContainerd `json:"containerdRegistryMirrors,omitempty"`
}

// ContainerRuntimeContainerd defines containerd container runtime registries configs.
type ContainerRuntimeContainerd struct {
	// A map of registries to use to render configs and mirrors for containerd registries
	Registries map[string]ContainerdRegistry `json:"registries,omitempty"`
}

// ContainerdRegistry defines endpoints and security for given container registry.
type ContainerdRegistry struct {
	// List of registry mirrors to use
	Mirrors []string `json:"mirrors,omitempty"`
}

// SeedMLASettings allow configuring seed level MLA (Monitoring, Logging & Alerting) stack settings.
type SeedMLASettings struct {
	// Optional: UserClusterMLAEnabled controls whether the user cluster MLA (Monitoring, Logging & Alerting) stack is enabled in the seed.
	UserClusterMLAEnabled bool `json:"userClusterMLAEnabled,omitempty"` //nolint:tagliatelle
}

// MeteringConfiguration contains all the configuration for the metering tool.
type MeteringConfiguration struct {
	Enabled bool `json:"enabled"`

	// StorageClassName is the name of the storage class that the metering prometheus instance uses to store metric data for reporting.
	StorageClassName string `json:"storageClassName"`
	// StorageSize is the size of the storage class. Default value is 100Gi.
	StorageSize string `json:"storageSize"`

	// +kubebuilder:default:={weekly: {schedule: "0 1 * * 6", interval: 7}}

	// ReportConfigurations is a map of report configuration definitions.
	ReportConfigurations map[string]*MeteringReportConfiguration `json:"reports,omitempty"`
}

type MeteringReportConfiguration struct {
	// +kubebuilder:default:=`0 1 * * 6`

	// Schedule in Cron format, see https://en.wikipedia.org/wiki/Cron. Please take a note that Schedule is responsible
	// only for setting the time when a report generation mechanism kicks off. The Interval MUST be set independently.
	Schedule string `json:"schedule,omitempty"`

	// +kubebuilder:default=7
	// +kubebuilder:validation:Minimum:=1

	// Interval defines the number of days consulted in the metering report.
	Interval uint32 `json:"interval,omitempty"`

	// +optional
	// +kubebuilder:validation:Minimum:=1

	// Retention defines a number of days after which reports are queued for removal. If not set, reports are kept forever.
	// Please note that this functionality works only for object storage that supports an object lifecycle management mechanism.
	Retention *uint32 `json:"retention,omitempty"`

	// +optional
	// +kubebuilder:default:={"cluster","namespace"}

	// Types of reports to generate. Available report types are cluster and namespace. By default, all types of reports are generated.
	Types []string `json:"type,omitempty"`
}

// IsDefaultEtcdAutomaticBackupEnabled returns true if etcd automatic backup is configured for the seed.
func (s *Seed) IsDefaultEtcdAutomaticBackupEnabled() bool {
	if cfg := s.Spec.EtcdBackupRestore; cfg != nil {
		return len(cfg.Destinations) > 0 && cfg.DefaultDestination != ""
	}

	return false
}

func (s *Seed) GetEtcdBackupDestination(destinationName string) *BackupDestination {
	if s.Spec.EtcdBackupRestore == nil {
		return nil
	}

	return s.Spec.EtcdBackupRestore.Destinations[destinationName]
}

// IsUpToDate returns true if the seed was successfully reconciled
// by the KKP operator, meaning it has the same CRDs and controller
// versions than the master.
func (s *Seed) IsUpToDate(masterVersions kubermatic.Versions) bool {
	return s.Status.Versions.Kubermatic == masterVersions.KubermaticCommit
}

func (s *Seed) SetKubermaticVersion(masterVersions kubermatic.Versions) {
	s.Status.Versions.Kubermatic = masterVersions.KubermaticCommit
}
