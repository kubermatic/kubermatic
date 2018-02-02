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

// DigitialoceanDatacenterSpec specifies a data center of digital ocean.
type DigitialoceanDatacenterSpec struct {
	Region string `json:"region"`
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
	AuthURL          string `json:"auth_url"`
}

// DatacenterSpec specifies the data for a datacenter.
type DatacenterSpec struct {
	Country      string                       `json:"country,omitempty"`
	Location     string                       `json:"location,omitempty"`
	Provider     string                       `json:"provider,omitempty"`
	Digitalocean *DigitialoceanDatacenterSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnDatacenterSpec  `json:"bringyourown,omitempty"`
	AWS          *AWSDatacenterSpec           `json:"aws,omitempty"`
	Openstack    *OpenstackDatacenterSpec     `json:"openstack,omitempty"`
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
	NodeControllerDeploymentYaml    string            `yaml:"nodeControllerDeploymentYaml"`
	AddonManagerDeploymentYaml      string            `yaml:"addonManagerDeploymentYaml"`
	MachineControllerDeploymentYaml string            `yaml:"machineControllerDeploymentYaml"`
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
