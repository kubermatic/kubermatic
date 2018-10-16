package v1

import (
	"time"

	"github.com/Masterminds/semver"

	apiv2 "github.com/kubermatic/kubermatic/api/pkg/api/v2"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"encoding/json"
	cmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

// LegacyObjectMeta is an object storing common metadata for persistable objects.
// Deprecated: LegacyObjectMeta is deprecated use ObjectMeta instead.
type LegacyObjectMeta struct {
	Name            string `json:"name"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
	UID             string `json:"uid,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// ObjectMeta defines the set of fields that objects returned from the API have
// swagger:model ObjectMeta
type ObjectMeta struct {
	// ID unique value that identifies the resource generated by the server
	ID string `json:"id"`

	// Name represents human readable name for the resource
	Name string `json:"name"`

	// DeletionTimestamp is a timestamp representing the server time when this object was deleted.
	DeletionTimestamp *time.Time `json:"deletionTimestamp,omitempty"`

	// CreationTimestamp is a timestamp representing the server time when this object was created.
	CreationTimestamp time.Time `json:"creationTimestamp,omitempty"`
}

// DigitialoceanDatacenterSpec specifies a datacenter of DigitalOcean.
type DigitialoceanDatacenterSpec struct {
	Region string `json:"region"`
}

// HetznerDatacenterSpec specifies a datacenter of Hetzner.
type HetznerDatacenterSpec struct {
	Datacenter string `json:"datacenter"`
	Location   string `json:"location"`
}

// ImageList defines a map of operating system and the image to use
type ImageList map[string]string

// VSphereDatacenterSpec specifies a datacenter of VSphere.
type VSphereDatacenterSpec struct {
	Endpoint   string    `json:"endpoint"`
	Datacenter string    `json:"datacenter"`
	Datastore  string    `json:"datastore"`
	Cluster    string    `json:"cluster"`
	Templates  ImageList `json:"templates"`
}

// BringYourOwnDatacenterSpec specifies a data center with bring-your-own nodes.
type BringYourOwnDatacenterSpec struct{}

// AWSDatacenterSpec specifies a data center of Amazon Web Services.
type AWSDatacenterSpec struct {
	Region string `json:"region"`
}

// AzureDatacenterSpec specifies a datacenter of Azure.
type AzureDatacenterSpec struct {
	Location string `json:"location"`
}

// OpenstackDatacenterSpec specifies a generic bare metal datacenter.
type OpenstackDatacenterSpec struct {
	AvailabilityZone string    `json:"availability_zone"`
	Region           string    `json:"region"`
	AuthURL          string    `json:"auth_url"`
	Images           ImageList `json:"images"`
}

// DatacenterSpec specifies the data for a datacenter.
type DatacenterSpec struct {
	Seed         string                       `json:"seed"`
	Country      string                       `json:"country,omitempty"`
	Location     string                       `json:"location,omitempty"`
	Provider     string                       `json:"provider,omitempty"`
	Digitalocean *DigitialoceanDatacenterSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnDatacenterSpec  `json:"bringyourown,omitempty"`
	AWS          *AWSDatacenterSpec           `json:"aws,omitempty"`
	Azure        *AzureDatacenterSpec         `json:"azure,omitempty"`
	Openstack    *OpenstackDatacenterSpec     `json:"openstack,omitempty"`
	Hetzner      *HetznerDatacenterSpec       `json:"hetzner,omitempty"`
	VSphere      *VSphereDatacenterSpec       `json:"vsphere,omitempty"`
}

// DatacenterList represents a list of datacenters
// swagger:model DatacenterList
type DatacenterList []Datacenter

// Datacenter is the object representing a Kubernetes infra datacenter.
// swagger:model Datacenter
type Datacenter struct {
	Metadata LegacyObjectMeta `json:"metadata"`
	Spec     DatacenterSpec   `json:"spec"`
	Seed     bool             `json:"seed,omitempty"`
}

// DigitaloceanSizeList represents a object of digitalocean sizes.
// swagger:model DigitaloceanSizeList
type DigitaloceanSizeList struct {
	Standard  []DigitaloceanSize `json:"standard"`
	Optimized []DigitaloceanSize `json:"optimized"`
}

// DigitaloceanSize is the object representing digitalocean sizes.
// swagger:model DigitaloceanSize
type DigitaloceanSize struct {
	Slug         string   `json:"slug"`
	Available    bool     `json:"available"`
	Transfer     float64  `json:"transfer"`
	PriceMonthly float64  `json:"price_monthly"`
	PriceHourly  float64  `json:"price_hourly"`
	Memory       int      `json:"memory"`
	VCPUs        int      `json:"vcpus"`
	Disk         int      `json:"disk"`
	Regions      []string `json:"regions"`
}

// AzureSizeList represents an array of Azure VM sizes.
// swagger:model AzureSizeList
type AzureSizeList []AzureSize

// AzureSize is the object representing Azure VM sizes.
// swagger:model AzureSize
type AzureSize struct {
	Name                 *string `json:"name"`
	NumberOfCores        *int32  `json:"numberOfCores"`
	OsDiskSizeInMB       *int32  `json:"osDiskSizeInMB"`
	ResourceDiskSizeInMB *int32  `json:"resourceDiskSizeInMB"`
	MemoryInMB           *int32  `json:"memoryInMB"`
	MaxDataDiskCount     *int32  `json:"maxDataDiskCount"`
}

// SSHKey represents a ssh key
// swagger:model SSHKey
type SSHKey struct {
	ObjectMeta
	Spec SSHKeySpec `json:"spec"`
}

// SSHKeySpec represents the details of a ssh key
type SSHKeySpec struct {
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"publicKey"`
}

// LegacyUser represents an API user that is used for authentication.
// Depreciated: use User instead
type LegacyUser struct {
	ID    string
	Name  string
	Email string
	Roles map[string]struct{}
}

// User represent an API user
// swagger:model User
type User struct {
	ObjectMeta

	// Email an email address of the user
	Email string `json:"email"`

	// Projects holds the list of project the user belongs to
	// along with the group names
	Projects []ProjectGroup `json:"projects,omitempty"`
}

// ProjectGroup is a helper data structure that
// stores the information about a project and a group prefix that a user belongs to
type ProjectGroup struct {
	ID          string `json:"id"`
	GroupPrefix string `json:"group"`
}

// Project is a top-level container for a set of resources
// swagger:model Project
type Project struct {
	ObjectMeta
	Status string `json:"status"`
}

// Kubeconfig is a clusters kubeconfig
// swagger:model Kubeconfig
type Kubeconfig struct {
	cmdv1.Config
}

// OpenstackSize is the object representing openstack's sizes.
// swagger:model OpenstackSize
type OpenstackSize struct {
	// Slug holds  the name of the size
	Slug string `json:"slug"`
	// Memory is the amount of memory, measured in MB
	Memory int `json:"memory"`
	// VCPUs indicates how many (virtual) CPUs are available for this flavor
	VCPUs int `json:"vcpus"`
	// Disk is the amount of root disk, measured in GB
	Disk int `json:"disk"`
	// Swap is the amount of swap space, measured in MB
	Swap int `json:"swap"`
	// Region specifies the geographic region in which the size resides
	Region string `json:"region"`
	// IsPublic indicates whether the size is public (available to all projects) or scoped to a set of projects
	IsPublic bool `json:"isPublic"`
}

// OpenstackSubnet is the object representing a openstack subnet.
// swagger:model OpenstackSubnet
type OpenstackSubnet struct {
	// Id uniquely identifies the subnet
	ID string `json:"id"`
	// Name is human-readable name for the subnet
	Name string `json:"name"`
}

// OpenstackTenant is the object representing a openstack tenant.
// swagger:model OpenstackTenant
type OpenstackTenant struct {
	// Id uniquely identifies the current tenant
	ID string `json:"id"`
	// Name is the name of the tenant
	Name string `json:"name"`
}

// OpenstackNetwork is the object representing a openstack network.
// swagger:model OpenstackNetwork
type OpenstackNetwork struct {
	// Id uniquely identifies the current network
	ID string `json:"id"`
	// Name is the name of the network
	Name string `json:"name"`
	// External set if network is the external network
	External bool `json:"external"`
}

// OpenstackSecurityGroup is the object representing a openstack security group.
// swagger:model OpenstackSecurityGroup
type OpenstackSecurityGroup struct {
	// Id uniquely identifies the current security group
	ID string `json:"id"`
	// Name is the name of the security group
	Name string `json:"name"`
}

// VSphereNetwork is the object representing a vsphere network.
// swagger:model VSphereNetwork
type VSphereNetwork struct {
	// Name is the name of the network
	Name string `json:"name"`
}

// MasterVersion describes a version of the master components
// swagger:model MasterVersion
type MasterVersion struct {
	Version             *semver.Version `json:"version"`
	AllowedNodeVersions []string        `json:"allowedNodeVersions"`
	Default             bool            `json:"default,omitempty"`
}

// Cluster defines the cluster resource
// swagger:model Cluster
type Cluster struct {
	ObjectMeta `json:",inline"`
	Spec       ClusterSpec   `json:"spec"`
	Status     ClusterStatus `json:"status"`
}

// ClusterSpec defines the cluster specification
type ClusterSpec struct {
	// Cloud specifies the cloud providers configuration
	Cloud kubermaticv1.CloudSpec `json:"cloud"`
	// MachineNetworks optionally specifies the parameters for IPAM.
	MachineNetworks []kubermaticv1.MachineNetworkingConfig `json:"machineNetworks,omitempty"`

	// Version desired version of the kubernetes master components
	Version string `json:"version"`
}

func (cs *NewClusterSpec) MarshalJSON() ([]byte, error) {
	ret, err := json.Marshal(struct {
		MachineNetworks []kubermaticv1.MachineNetworkingConfig `json:"machineNetworks,omitempty"`
		Version         string                                 `json:"version"`

		// Overshadows kubermaticv1.CloudSpec to avoid putting sensitive data in API responses.
		Cloud PublicCloudSpec `json:"cloud"`
	}{
		Cloud: PublicCloudSpec{
			DatacenterName: cs.Cloud.DatacenterName,
			Fake:           NewPublicFakeCloudSpec(cs.Cloud.Fake),
			Digitalocean:   NewPublicDigitaloceanCloudSpec(cs.Cloud.Digitalocean),
			BringYourOwn:   NewPublicBringYourOwnCloudSpec(cs.Cloud.BringYourOwn),
			AWS:            NewPublicAWSCloudSpec(cs.Cloud.AWS),
			Azure:          NewPublicAzureCloudSpec(cs.Cloud.Azure),
			Openstack:      NewPublicOpenstackCloudSpec(cs.Cloud.Openstack),
			Hetzner:        NewPublicHetznerCloudSpec(cs.Cloud.Hetzner),
			VSphere:        NewPublicVSphereCloudSpec(cs.Cloud.VSphere),
		},
		Version:         cs.Version,
		MachineNetworks: cs.MachineNetworks,
	})

	return ret, err
}

// NewCloudSpec overshadows kubermaticv1.CloudSpec to avoid putting sensitive data in API responses.
type PublicCloudSpec struct {
	DatacenterName string                       `json:"dc"`
	Fake           *PublicFakeCloudSpec         `json:"fake,omitempty"`
	Digitalocean   *PublicDigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	BringYourOwn   *PublicBringYourOwnCloudSpec `json:"bringyourown,omitempty"`
	AWS            *PublicAWSCloudSpec          `json:"aws,omitempty"`
	Azure          *PublicAzureCloudSpec        `json:"azure,omitempty"`
	Openstack      *PublicOpenstackCloudSpec    `json:"openstack,omitempty"`
	Hetzner        *PublicHetznerCloudSpec      `json:"hetzner,omitempty"`
	VSphere        *PublicVSphereCloudSpec      `json:"vsphere,omitempty"`
}

type PublicFakeCloudSpec struct{}

func NewPublicFakeCloudSpec(internal *kubermaticv1.FakeCloudSpec) (public *PublicFakeCloudSpec) {
	if internal == nil {
		return nil
	}

	return &PublicFakeCloudSpec{}
}

type PublicDigitaloceanCloudSpec struct{}

func NewPublicDigitaloceanCloudSpec(internal *kubermaticv1.DigitaloceanCloudSpec) (public *PublicDigitaloceanCloudSpec) {
	if internal == nil {
		return nil
	}

	return &PublicDigitaloceanCloudSpec{}
}

type PublicHetznerCloudSpec struct{}

func NewPublicHetznerCloudSpec(internal *kubermaticv1.HetznerCloudSpec) (public *PublicHetznerCloudSpec) {
	if internal == nil {
		return nil
	}

	return &PublicHetznerCloudSpec{}
}

type PublicAzureCloudSpec struct {
	TenantID        string `json:"tenantID"`
	SubscriptionID  string `json:"subscriptionID"`
	ResourceGroup   string `json:"resourceGroup"`
	VNetName        string `json:"vnet"`
	SubnetName      string `json:"subnet"`
	RouteTableName  string `json:"routeTable"`
	SecurityGroup   string `json:"securityGroup"`
	AvailabilitySet string `json:"availabilitySet"`
}

func NewPublicAzureCloudSpec(internal *kubermaticv1.AzureCloudSpec) (public *PublicAzureCloudSpec) {
	if internal == nil {
		return nil
	}

	return &PublicAzureCloudSpec{
		TenantID:        internal.TenantID,
		SubscriptionID:  internal.SubscriptionID,
		ResourceGroup:   internal.ResourceGroup,
		VNetName:        internal.VNetName,
		SubnetName:      internal.SubnetName,
		AvailabilitySet: internal.AvailabilitySet,
		RouteTableName:  internal.RouteTableName,
		SecurityGroup:   internal.SecurityGroup,
	}
}

type PublicVSphereCloudSpec struct {
	VMNetName string `json:"vmNetName"`
}

func NewPublicVSphereCloudSpec(internal *kubermaticv1.VSphereCloudSpec) (public *PublicVSphereCloudSpec) {
	if internal == nil {
		return nil
	}

	return &PublicVSphereCloudSpec{
		VMNetName: internal.VMNetName,
	}
}

type PublicBringYourOwnCloudSpec struct{}

func NewPublicBringYourOwnCloudSpec(internal *kubermaticv1.BringYourOwnCloudSpec) (public *PublicBringYourOwnCloudSpec) {
	if internal == nil {
		return nil
	}

	return &PublicBringYourOwnCloudSpec{}
}

type PublicAWSCloudSpec struct {
	VPCID               string `json:"vpcId"`
	SubnetID            string `json:"subnetId"`
	RoleName            string `json:"roleName"`
	RouteTableID        string `json:"routeTableId"`
	InstanceProfileName string `json:"instanceProfileName"`
	SecurityGroupID     string `json:"securityGroupID"`
	AvailabilityZone    string `json:"availabilityZone"`
}

func NewPublicAWSCloudSpec(internal *kubermaticv1.AWSCloudSpec) (public *PublicAWSCloudSpec) {
	if internal == nil {
		return nil
	}

	return &PublicAWSCloudSpec{
		VPCID:               internal.VPCID,
		SubnetID:            internal.SubnetID,
		RoleName:            internal.RoleName,
		RouteTableID:        internal.RouteTableID,
		InstanceProfileName: internal.InstanceProfileName,
		SecurityGroupID:     internal.SecurityGroupID,
		AvailabilityZone:    internal.AvailabilityZone,
	}
}

type PublicOpenstackCloudSpec struct {
	Tenant         string `json:"tenant"`
	Domain         string `json:"domain"`
	Network        string `json:"network"`
	SecurityGroups string `json:"securityGroups"`
	FloatingIPPool string `json:"floatingIpPool"`
	RouterID       string `json:"routerID"`
	SubnetID       string `json:"subnetID"`
}

func NewPublicOpenstackCloudSpec(internal *kubermaticv1.OpenstackCloudSpec) (public *PublicOpenstackCloudSpec) {
	if internal == nil {
		return nil
	}

	return &PublicOpenstackCloudSpec{
		Tenant:         internal.Tenant,
		Domain:         internal.Domain,
		Network:        internal.Network,
		SecurityGroups: internal.SecurityGroups,
		FloatingIPPool: internal.FloatingIPPool,
		RouterID:       internal.RouterID,
		SubnetID:       internal.SubnetID,
	}
}

// ClusterStatus defines the cluster status
type ClusterStatus struct {
	// Version actual version of the kubernetes master components
	Version string `json:"version"`

	// URL specifies the address at which the cluster is available
	URL string `json:"url"`
}

// ClusterHealth stores health information about the cluster's components.
// swagger:model ClusterHealth
type ClusterHealth struct {
	Apiserver         bool `json:"apiserver"`
	Scheduler         bool `json:"scheduler"`
	Controller        bool `json:"controller"`
	MachineController bool `json:"machineController"`
	Etcd              bool `json:"etcd"`
}

// ClusterList represents a list of clusters
// swagger:model ClusterList
type ClusterList []Cluster

// Node represents a worker node that is part of a cluster
// swagger:model Node
type Node struct {
	ObjectMeta `json:",inline"`

	// TODO: normally referring to a field that is defined in v2 is bad, if you are doing this please stop
	// TODO: I did this only because I know that we are working on the user management
	// TODO: and once we have it then we will remove apiv2.LegacyNode struct.
	Spec apiv2.NodeSpec `json:"spec"`

	// TODO: normally referring to a field that is defined in v2 is bad, if you are doing this please stop
	// TODO: I did this only because I know that we are working on the user management
	// TODO: and once we have it then we will remove apiv2.LegacyNode struct.
	Status apiv2.NodeStatus `json:"status"`
}

// ClusterMetric defines a metric for the given cluster
// swagger:model ClusterMetric
type ClusterMetric struct {
	Name   string    `json:"name"`
	Values []float64 `json:"values,omitempty"`
}
