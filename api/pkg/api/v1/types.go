package v1

import (
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	corev1 "k8s.io/api/core/v1"
	cmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

// ObjectMeta is an object storing common metadata for persistable objects.
type ObjectMeta struct {
	Name            string `json:"name"`
	ResourceVersion string `json:"resourceVersion,omitempty"`
	UID             string `json:"uid,omitempty"`

	Annotations map[string]string `json:"annotations,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
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

// VSphereDatacenterSpec specifies a datacenter of VSphere.
type VSphereDatacenterSpec struct {
	Endpoint   string `json:"endpoint"`
	Datacenter string `json:"datacenter"`
	Datastore  string `json:"datastore"`
	Cluster    string `json:"cluster"`
}

// BringYourOwnDatacenterSpec specifies a data center with bring-your-own nodes.
type BringYourOwnDatacenterSpec struct{}

// AWSDatacenterSpec specifies a data center of Amazon Web Services.
type AWSDatacenterSpec struct {
	Region string `json:"region"`
}

// OpenstackDatacenterSpec specifies a generic bare metal datacenter.
type OpenstackDatacenterSpec struct {
	AvailabilityZone string `json:"availability_zone"`
	Region           string `json:"region"`
	AuthURL          string `json:"auth_url"`
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
	Metadata ObjectMeta     `json:"metadata"`
	Spec     DatacenterSpec `json:"spec"`
	Seed     bool           `json:"seed,omitempty"`
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

// MasterVersion is the object representing a Kubernetes Master version.
// swagger:model MasterVersion
type MasterVersion struct {
	Name                            string            `yaml:"name"`
	ID                              string            `yaml:"id"`
	Default                         bool              `yaml:"default"`
	AllowedNodeVersions             []string          `yaml:"allowedNodeVersions"`
	EtcdOperatorDeploymentYaml      string            `yaml:"etcdOperatorDeploymentYaml"`
	EtcdClusterYaml                 string            `yaml:"etcdClusterYaml"`
	ApiserverDeploymentYaml         string            `yaml:"apiserverDeploymentYaml"`
	ControllerDeploymentYaml        string            `yaml:"controllerDeploymentYaml"`
	SchedulerDeploymentYaml         string            `yaml:"schedulerDeploymentYaml"`
	AddonManagerDeploymentYaml      string            `yaml:"addonManagerDeploymentYaml"`
	MachineControllerDeploymentYaml string            `yaml:"machineControllerDeploymentYaml"`
	KubeStateMetricsDeploymentYaml  string            `yaml:"kubeStateMetricsDeploymentYaml"`
	OpenVPNServerDeploymentYaml     string            `yaml:"openVPNServerDeploymentYaml"`
	Values                          map[string]string `yaml:"values"`
}

// NodeVersion is the object representing a Kubernetes Kubelet version.
type NodeVersion struct {
	Name, ID string
	Latest   bool
}

// MasterUpdate represents an update option for K8s master components
type MasterUpdate struct {
	From            string `yaml:"from"`
	To              string `yaml:"to"`
	Automatic       bool   `yaml:"automatic"`
	RollbackAllowed bool   `yaml:"rollbackAllowed"`
	Enabled         bool   `yaml:"enabled"`
	Visible         bool   `yaml:"visible"`
	Promote         bool   `yaml:"promote"`
}

// NodeUpdate represents an update option for K8s node components
type NodeUpdate struct {
	From, To                   string
	Automatic, RollbackAllowed bool
	Enabled                    bool
	Visible                    bool
	Promote                    bool
}

// SSHKey represents a ssh key
// swagger:model SSHKey
type SSHKey struct {
	Metadata ObjectMeta `json:"metadata"`
	Spec     SSHKeySpec `json:"spec"`
}

// SSHKeySpec represents the details of a ssh key
type SSHKeySpec struct {
	Owner       string   `json:"owner"`
	Name        string   `json:"name"`
	Fingerprint string   `json:"fingerprint"`
	PublicKey   string   `json:"publicKey"`
	Clusters    []string `json:"clusters"`
}

// User represents an API user that is used for authentication.
type User struct {
	ID    string
	Name  string
	Email string
	Roles map[string]struct{}
}

// Kubeconfig is a clusters kubeconfig
// swagger:model Kubeconfig
type Kubeconfig struct {
	cmdv1.Config
}

// ClusterList represents a list of clusters
// swagger:model ClusterListV1
type ClusterList []Cluster

// Cluster is the object representing a cluster.
// swagger:model ClusterV1
type Cluster struct {
	kubermaticv1.Cluster
}

// NodeList represents a list of nodes
// swagger:model NodeListV1
type NodeList []Node

// Node is the object representing a cluster node.
// swagger:model NodeV1
type Node struct {
	corev1.Node
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

// Addon is the API representation of a cluster addon, a user could install
type Addon struct {
	Name string `json:"name"`
}
