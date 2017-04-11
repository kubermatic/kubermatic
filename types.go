package api

import (
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	cmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	cmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
)

// Metadata is an object storing common metadata for persistable objects.
type Metadata struct {
	Name     string `json:"name"`
	Revision string `json:"revision,omitempty"`
	UID      string `json:"uid,omitempty"`

	// private fields
	// Annotations represent Annotations on Kubernetes Namespace for the respective cluster,
	// which are used to store persistent data for the cluster.
	Annotations map[string]string `json:"-"`
	User        string            `json:"-"`
}

// DigitaloceanNodeSpec specifies a digital ocean node.
type DigitaloceanNodeSpec struct {
	// Type specifies the name of the image used to create the node.
	Type string `json:"type"`
	// Size is the size of the node (DigitalOcean node type).
	Size string `json:"size"`
	// SSHKeyFingerprints  represent the fingerprints of the keys.
	// DigitalOcean utilizes the fingerprints to identify public
	// SSHKeys stored within the DigitalOcean platform.
	SSHKeyFingerprints []string `json:"sshKeys,omitempty"`
}

// BringYourOwnNodeSpec specifies a bring your own node
type BringYourOwnNodeSpec struct {
}

// BareMetalNodeSpec specifies a node instanciated by the bare-metal-provider
type BareMetalNodeSpec struct {
	ID       string `json:"id"`
	Memory   int    `json:"memory"`
	Space    int    `json:"space"`
	CPUs     []CPU  `json:"cpus"`
	PublicIP string `json:"public_ip"`
}

// FakeNodeSpec specifies a fake node.
type FakeNodeSpec struct {
	Type string `json:"type"`
	OS   string `json:"os"`
}

// AWSNodeSpec specifies an aws node.
type AWSNodeSpec struct {
	Type           string                 `json:"type"`
	DiskSize       int64                  `json:"disk_size"`
	ContainerLinux ContainerLinuxNodeSpec `json:"container_linux"`
}

// NodeSpec mutually stores data of a cloud specific node.
type NodeSpec struct {
	// DatacenterName contains the name of the datacenter the node is located in.
	DatacenterName string `json:"dc"`

	Digitalocean *DigitaloceanNodeSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnNodeSpec `json:"bringyourown,omitempty"`
	Fake         *FakeNodeSpec         `json:"fake,omitempty"`
	AWS          *AWSNodeSpec          `json:"aws,omitempty"`
	BareMetal    *BareMetalNodeSpec    `json:"baremetal,omitempty"`
}

// NodeCondition stores information about the node condition
type NodeCondition struct {
	Healthy     bool   `json:"healthy"`
	Description string `json:"description"`
}

// NodeStatus stores status information about a node.
type NodeStatus struct {
	Addresses NodeAddresses `json:"addresses"`
	CPU       int64         `json:"cpu"`
	Memory    string        `json:"memory"`
	Versions  *NodeVersions `json:"versions"`
	Condition NodeCondition `json:"condition"`
}

// NodeAddresses stores the IP addresses associated with a node
type NodeAddresses struct {
	Public  string `json:"public"`
	Private string `json:"private"`
}

// NodeVersions stores information about the node operating system
type NodeVersions struct {
	OS               string `json:"os,omitempty"`
	ContainerRuntime string `json:"container_runtime,omitempty"`
	Kubelet          string `json:"kubelet,omitempty"`
	KubeProxy        string `json:"kubeproxy,omitempty"`
	Kernel           string `json:"kernel,omitempty"`
}

// Node is the object representing a cluster node.
type Node struct {
	Metadata Metadata   `json:"metadata"`
	Spec     NodeSpec   `json:"spec"`
	Status   NodeStatus `json:"status,omitempty"`
}

// DigitaloceanCloudSpec specifies access data to digital ocean.
type DigitaloceanCloudSpec struct {
	// APIToken is used to authenticate with the DigitalOcean API.
	Token string `json:"token"`
	// SSHKeys are SSH keys used in the cloud-init generation to deploy to nodes.
	SSHKeys []string `json:"sshKeys"`
}

// ContainerLinuxClusterSpec specifies container linux configuration for nodes - cluster wide
type ContainerLinuxClusterSpec struct {
	AutoUpdate bool `json:"auto_update"`
}

// ContainerLinuxNodeSpec specifies container linux configuration for individual nodes
type ContainerLinuxNodeSpec struct {
	Version string `json:"version"`
}

// AWSCloudSpec specifies access data to Amazon Web Services.
type AWSCloudSpec struct {
	AccessKeyID         string                    `json:"access_key_id"`
	SecretAccessKey     string                    `json:"secret_access_key"`
	VPCId               string                    `json:"vpc_id"`
	SSHKeyName          string                    `json:"ssh_key_name"`
	SubnetID            string                    `json:"subnet_id"`
	InternetGatewayID   string                    `json:"internet_gateway_id"`
	RouteTableID        string                    `json:"route_table_id"`
	RoleName            string                    `json:"role_name"`
	PolicyName          string                    `json:"policy"`
	InstanceProfileName string                    `json:"instance_profile_name"`
	AvailabilityZone    string                    `json:"availability_zone"`
	SecurityGroupID     string                    `json:"security_group_id"`
	ContainerLinux      ContainerLinuxClusterSpec `json:"container_linux"`
}

// BringYourOwnCloudSpec specifies access data for a bring your own cluster.
type BringYourOwnCloudSpec struct {
	PrivateIntf   string  `json:"privateInterface"`
	ClientKeyCert KeyCert `json:"clientKeyCert"`
}

// BareMetalCloudSpec specifies access to a bare metal datacenter
type BareMetalCloudSpec struct {
	Name string `json:"name"`
}

// FakeCloudSpec specifies access data for a fake cloud.
type FakeCloudSpec struct {
	Token string `json:"token,omitempty"`
}

// FlannelNetworkSpec specifies a deployed flannel network.
type FlannelNetworkSpec struct {
	// CIDR is the subnet used by Flannel in CIDR notation.
	// See RFC 4632, e.g. "127.1.0.0/16"
	CIDR string
}

// NetworkSpec specifies the deployed network.
type NetworkSpec struct {
	// FlannelNetworkSpec holds the required information to configure Flannel
	Flannel FlannelNetworkSpec
}

// CloudSpec mutually stores access data to a cloud provider.
type CloudSpec struct {
	// DatacenterName where the users 'cloud' lives in.
	DatacenterName string `json:"dc"`
	// Network holds the network specification object.
	Network NetworkSpec `json:"-"`

	User   string `json:"user,omitempty"`
	Secret string `json:"secret,omitempty"`
	Name   string `json:"name,omitempty"`
	Region string `json:"region,omitempty"`

	Fake         *FakeCloudSpec         `json:"fake,omitempty"`
	Digitalocean *DigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnCloudSpec `json:"bringyourown,omitempty"`
	AWS          *AWSCloudSpec          `json:"aws,omitempty"`
	BareMetal    *BareMetalCloudSpec    `json:"baremetal,omitempty"`
}

// ClusterHealthStatus stores health information of the components of a cluster.
type ClusterHealthStatus struct {
	Apiserver  bool   `json:"apiserver"`
	Scheduler  bool   `json:"scheduler"`
	Controller bool   `json:"controller"`
	Etcd       []bool `json:"etcd"`
}

// ClusterHealth stores health information of a cluster and the timestamp of the last change.
type ClusterHealth struct {
	ClusterHealthStatus `json:",inline"`
	LastTransitionTime  time.Time `json:"lastTransitionTime"`
}

// ClusterPhase is the life cycle phase of a cluster.
type ClusterPhase string

const (
	// UnknownClusterStatusPhase means that the phase label is missing on the Namespace.
	UnknownClusterStatusPhase ClusterPhase = "Unknown"

	// PendingClusterStatusPhase means that the cluster controller hasn't picked the cluster up.
	PendingClusterStatusPhase ClusterPhase = "Pending"

	// LaunchingClusterStatusPhase means that the cluster controller starts up the cluster.
	LaunchingClusterStatusPhase ClusterPhase = "Launching"

	// FailedClusterStatusPhase means that the cluster controller time out launching the cluster.
	FailedClusterStatusPhase ClusterPhase = "Failed"

	// RunningClusterStatusPhase means that the cluster is cluster is up and running.
	RunningClusterStatusPhase ClusterPhase = "Running"

	// PausedClusterStatusPhase means that the cluster was paused after the idle time.
	PausedClusterStatusPhase ClusterPhase = "Paused"

	// DeletingClusterStatusPhase means that the cluster controller is deleting the cluster.
	DeletingClusterStatusPhase ClusterPhase = "Deleting"

	// UpdatingMasterClusterStatusPhase means that the cluster controller is updating the master components of the cluster.
	UpdatingMasterClusterStatusPhase ClusterPhase = "Updatingmaster"

	// UpdatingNodesClusterStatusPhase means that the cluster controller is updating the nodes of the cluster.
	UpdatingNodesClusterStatusPhase ClusterPhase = "Updatingnodes"
)

// MasterUpdatePhase represents the current master update phase.
type MasterUpdatePhase string

const (
	// StartMasterUpdatePhase means that the update controller is updating etcd operator.
	StartMasterUpdatePhase MasterUpdatePhase = "Starting"

	// EtcdOperatorUpdatePhase means that the update controller is waiting for etcd operator and updating the etcd cluster.
	EtcdOperatorUpdatePhase MasterUpdatePhase = "WaitEtcdOperatorReady"

	// EtcdClusterUpdatePhase means that the update controller is waiting for etcd cluster and updating the API server.
	EtcdClusterUpdatePhase MasterUpdatePhase = "WaitEtcdReady"

	// APIServerMasterUpdatePhase means that the update controller is waiting for the apiserver and updating the controllers.
	APIServerMasterUpdatePhase MasterUpdatePhase = "WaitAPIReady"

	// ControllersMasterUpdatePhase means that the update controller is waiting for the controllers.
	ControllersMasterUpdatePhase MasterUpdatePhase = "WaitControllersReady"

	// FinishMasterUpdatePhase means that the update controller has finished the update.
	FinishMasterUpdatePhase MasterUpdatePhase = "Finished"
)

type (
	// Bytes stores a byte slices and ecnodes as base64 in JSON.
	Bytes []byte
)

// KeyCert is a pair of key and cert.
type KeyCert struct {
	Key  Bytes `json:"key"`
	Cert Bytes `json:"cert"`
}

// SecretKeyCert is a pair of key and cert where the key is not published to the API client.
type SecretKeyCert struct {
	Key  Bytes `json:"-"`
	Cert Bytes `json:"cert"`
}

// CPU represents the CPU resources available on a node
type CPU struct {
	Cores     int     `json:"cores"`
	Frequency float64 `json:"frequency"`
}

// ClusterStatus stores status information about a cluster.
type ClusterStatus struct {
	LastTransitionTime        time.Time         `json:"lastTransitionTime"`
	Phase                     ClusterPhase      `json:"phase,omitempty"`
	Health                    *ClusterHealth    `json:"health,omitempty"`
	LastDeployedMasterVersion string            `json:"lastDeployedMasterVersion"`
	MasterUpdatePhase         MasterUpdatePhase `json:"masterUpdatePhase"`

	RootCA       SecretKeyCert `json:"rootCA"`
	ApiserverSSH string        `json:"apiserverSSH"`
}

// SSHKeyFingerprint represents a representation to a extensions.UserSSHKey
type SSHKeyFingerprint string

// Equal takes a fingerprint and tests it equality
func (s SSHKeyFingerprint) Equal(c string) bool {
	// This is really slow :(
	a := strings.Replace(c, ":", "", -1)
	b := strings.Replace(string(s), ":", "", -1)
	return strings.EqualFold(a, b)
}

// ClusterSpec specifies the data for a new cluster.
type ClusterSpec struct {
	Cloud *CloudSpec `json:"cloud,omitempty"`
	// HumanReadableName is the cluster name provided by the user
	HumanReadableName string `json:"humanReadableName"`
	MasterVersion     string `json:"masterVersion"`

	SSHKeys []SSHKeyFingerprint `json:"ssh_fingerprints,omitempty"`

	Dev bool `json:"-"` // a cluster used in development, compare --dev flag.
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	URL      string `json:"url"`
	EtcdURL  string `json:"etcdURL"`
	Token    string `json:"token"`
	NodePort int    `json:"nodePort"`
}

// Cluster is the object representing a cluster.
type Cluster struct {
	Metadata Metadata        `json:"metadata"`
	Spec     ClusterSpec     `json:"spec"`
	Address  *ClusterAddress `json:"address,omitempty"`
	Status   ClusterStatus   `json:"status,omitempty"`
}

// GetKubeconfig returns a kubeconfig to connect to the cluster
func (c *Cluster) GetKubeconfig() *cmdv1.Config {
	return &cmdv1.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		CurrentContext: c.Metadata.Name,
		Clusters: []cmdv1.NamedCluster{{
			Name: c.Metadata.Name,
			Cluster: cmdv1.Cluster{
				Server: c.Address.URL,
				CertificateAuthorityData: c.Status.RootCA.Cert,
			},
		}},
		Contexts: []cmdv1.NamedContext{{
			Name: c.Metadata.Name,
			Context: cmdv1.Context{
				Cluster:  c.Metadata.Name,
				AuthInfo: c.Metadata.Name,
			},
		}},
		AuthInfos: []cmdv1.NamedAuthInfo{{
			Name: c.Metadata.Name,
			AuthInfo: cmdv1.AuthInfo{
				Token: c.Address.Token,
			},
		}},
	}
}

func (c *Cluster) getClientConfig() (clientcmd.ClientConfig, error) {
	v1cfg := c.GetKubeconfig()
	oldCfg := &cmdapi.Config{}
	err := latest.Scheme.Convert(v1cfg, oldCfg, nil)
	if err != nil {
		return nil, err
	}
	return clientcmd.NewNonInteractiveClientConfig(
		*oldCfg,
		v1cfg.Contexts[0].Name,
		&clientcmd.ConfigOverrides{},
		nil,
	), nil
}

// GetClient returns a kubernetes client which speaks to the cluster
func (c *Cluster) GetClient() (*kubernetes.Clientset, error) {
	cfg, err := c.getClientConfig()
	if err != nil {
		return nil, err
	}

	ccfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(ccfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// DigitialoceanDatacenterSpec specifies a data center of digital ocean.
type DigitialoceanDatacenterSpec struct {
	Region string `json:"region"`
}

// BringYourOwnDatacenterSpec specifies a data center with bring-your-own nodes.
type BringYourOwnDatacenterSpec struct {
}

// AWSDatacenterSpec specifies a data center of Amazon Web Services.
type AWSDatacenterSpec struct {
	Region string `json:"region"`
}

// BareMetalDatacenterSpec specifies a generic bare metal datacenter.
type BareMetalDatacenterSpec struct {
}

// DatacenterSpec specifies the data for a datacenter.
type DatacenterSpec struct {
	Country      string                       `json:"country,omitempty"`
	Location     string                       `json:"location,omitempty"`
	Provider     string                       `json:"provider,omitempty"`
	Digitalocean *DigitialoceanDatacenterSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnDatacenterSpec  `json:"bringyourown,omitempty"`
	AWS          *AWSDatacenterSpec           `json:"aws,omitempty"`
	BareMetal    *BareMetalDatacenterSpec     `json:"baremetal,omitempty"`
}

// Datacenter is the object representing a Kubernetes infra datacenter.
type Datacenter struct {
	Metadata Metadata       `json:"metadata"`
	Spec     DatacenterSpec `json:"spec"`
	Seed     bool           `json:"seed,omitempty"`
}

// MasterVersion is the object representing a Kubernetes Master version.
type MasterVersion struct {
	Name                       string            `yaml:"name"`
	ID                         string            `yaml:"id"`
	Default                    bool              `yaml:"default"`
	AllowedNodeVersions        []string          `yaml:"allowedNodeVersions"`
	EtcdOperatorDeploymentYaml string            `yaml:"etcdOperatorDeploymentYaml"`
	EtcdClusterYaml            string            `yaml:"etcdClusterYaml"`
	ApiserverDeploymentYaml    string            `yaml:"apiserverDeploymentYaml"`
	ControllerDeploymentYaml   string            `yaml:"controllerDeploymentYaml"`
	SchedulerDeploymentYaml    string            `yaml:"schedulerDeploymentYaml"`
	Values                     map[string]string `yaml:"values"`
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
	Visable                    bool
	Promote                    bool
}
