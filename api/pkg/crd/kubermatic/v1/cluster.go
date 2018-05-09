package v1

import (
	"encoding/base64"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ClusterPlural = "clusters"
)

// ClusterPhase is the life cycle phase of a cluster.
type ClusterPhase string

const (
	// NoneClusterStatusPhase is an not assigned cluster phase, the controller will assign a default.
	NoneClusterStatusPhase ClusterPhase = ""

	// ValidatingClusterStatusPhase means that the cluster will be verified.
	ValidatingClusterStatusPhase ClusterPhase = "Validating"

	// LaunchingClusterStatusPhase means that the cluster controller starts up the cluster.
	LaunchingClusterStatusPhase ClusterPhase = "Launching"

	// RunningClusterStatusPhase means that the cluster is cluster is up and running.
	RunningClusterStatusPhase ClusterPhase = "Running"

	// DeletingClusterStatusPhase means that the cluster controller is deleting the cluster.
	DeletingClusterStatusPhase ClusterPhase = "Deleting"
)

var ClusterPhases = []ClusterPhase{
	NoneClusterStatusPhase,
	ValidatingClusterStatusPhase,
	LaunchingClusterStatusPhase,
	RunningClusterStatusPhase,
	DeletingClusterStatusPhase,
}

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

const (
	WorkerNameLabelKey = "worker-name"
)

//+genclient
//+genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the object representing a cluster.
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec    ClusterSpec     `json:"spec"`
	Address *ClusterAddress `json:"address,omitempty"`
	Status  ClusterStatus   `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList specifies a list of clusters
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Cluster `json:"items"`
}

// ClusterSpec specifies the data for a new cluster.
type ClusterSpec struct {
	Cloud          *CloudSpec              `json:"cloud"`
	ClusterNetwork ClusterNetworkingConfig `json:"clusterNetwork"`

	HumanReadableName string `json:"humanReadableName"` // HumanReadableName is the cluster name provided by the user
	MasterVersion     string `json:"masterVersion"`
	WorkerName        string `json:"workerName"` // WorkerName is a cluster used in development, compare --worker-name flag.
	// Pause tells that this cluster is currently not managed by the controller.
	// It indicates that the user needs to do some action to resolve the pause.
	Pause bool `json:"pause"`
	// PauseReason is the reason why the cluster is no being managed.
	PauseReason string `json:"pauseReason,omitempty"`
}

// ClusterNetworkingConfig specifies the different networking
// parameters for a cluster.
type ClusterNetworkingConfig struct {
	// The network ranges from which service VIPs are allocated.
	Services NetworkRanges `json:"services"`

	// The network ranges from which POD networks are allocated.
	Pods NetworkRanges `json:"pods"`

	// Domain name for services.
	DNSDomain string `json:"dnsDomain"`
}

// NetworkRanges represents ranges of network addresses.
type NetworkRanges struct {
	CIDRBlocks []string `json:"cidrBlocks"`
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	URL          string `json:"url"`
	ExternalName string `json:"externalName"`
	KubeletToken string `json:"kubeletToken"`
	AdminToken   string `json:"adminToken"`
	IP           string `json:"ip"`
}

// ClusterStatus stores status information about a cluster.
type ClusterStatus struct {
	LastUpdated               metav1.Time       `json:"lastUpdated,omitempty"`
	Phase                     ClusterPhase      `json:"phase,omitempty"`
	Health                    *ClusterHealth    `json:"health,omitempty"`
	LastDeployedMasterVersion string            `json:"lastDeployedMasterVersion"`
	MasterUpdatePhase         MasterUpdatePhase `json:"masterUpdatePhase"`

	RootCA            KeyCert `json:"rootCA"`
	ApiserverCert     KeyCert `json:"apiserverCert"`
	KubeletCert       KeyCert `json:"kubeletCert"`
	ApiserverSSHKey   RSAKeys `json:"apiserverSshKey"`
	ServiceAccountKey Bytes   `json:"serviceAccountKey"`
	NamespaceName     string  `json:"namespaceName"`

	UserName  string `json:"userName"`
	UserEmail string `json:"userEmail"`

	ErrorReason  *ClusterStatusError `json:"errorReason,omitempty"`
	ErrorMessage *string             `json:"errorMessage,omitempty"`
}

type ClusterStatusError string

const (
	InvalidConfigurationClusterError ClusterStatusError = "InvalidConfiguration"
	UnsupportedChangeClusterError    ClusterStatusError = "UnsupportedChange"
	ReconcileClusterError            ClusterStatusError = "ReconcileError"
)

// CloudSpec mutually stores access data to a cloud provider.
type CloudSpec struct {
	// DatacenterName where the users 'cloud' lives in.
	DatacenterName string `json:"dc"`

	Fake         *FakeCloudSpec         `json:"fake,omitempty"`
	Digitalocean *DigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnCloudSpec `json:"bringyourown,omitempty"`
	AWS          *AWSCloudSpec          `json:"aws,omitempty"`
	Openstack    *OpenstackCloudSpec    `json:"openstack,omitempty"`
	Hetzner      *HetznerCloudSpec      `json:"hetzner,omitempty"`
	VSphere      *VSphereCloudSpec      `json:"vsphere,omitempty"`
}

// ClusterHealth stores health information of a cluster and the timestamp of the last change.
type ClusterHealth struct {
	ClusterHealthStatus `json:",inline"`
	LastTransitionTime  metav1.Time `json:"lastTransitionTime"`
}

// KeyCert is a pair of key and cert.
type KeyCert struct {
	Key  Bytes `json:"key"`
	Cert Bytes `json:"cert"`
}

// RSAKeys is a pair of private and public key where the key is not published to the API client.
type RSAKeys struct {
	PrivateKey Bytes `json:"privateKey"`
	PublicKey  Bytes `json:"publicKey"`
}

type Bytes []byte

// FakeCloudSpec specifies access data for a fake cloud.
type FakeCloudSpec struct {
	Token string `json:"token,omitempty"`
}

// DigitaloceanCloudSpec specifies access data to DigitalOcean.
type DigitaloceanCloudSpec struct {
	Token string `json:"token"` // Token is used to authenticate with the DigitalOcean API.
}

// HetznerCloudSpec specifies access data to hetzner cloud.
type HetznerCloudSpec struct {
	Token string `json:"token"` // Token is used to authenticate with the Hetzner cloud API.
}

// VSphereCloudSpec specifies access data to VSphere cloud.
type VSphereCloudSpec struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// BringYourOwnCloudSpec specifies access data for a bring your own cluster.
type BringYourOwnCloudSpec struct{}

// AWSCloudSpec specifies access data to Amazon Web Services.
type AWSCloudSpec struct {
	AccessKeyID         string `json:"accessKeyId"`
	SecretAccessKey     string `json:"secretAccessKey"`
	VPCID               string `json:"vpcId"`
	SubnetID            string `json:"subnetId"`
	RoleName            string `json:"roleName"`
	RouteTableID        string `json:"routeTableId"`
	InstanceProfileName string `json:"instanceProfileName"`
	SecurityGroup       string `json:"securityGroup"`
	SecurityGroupID     string `json:"securityGroupID"`

	AvailabilityZone string `json:"availabilityZone"`
}

// OpenstackCloudSpec specifies access data to an openstack cloud.
type OpenstackCloudSpec struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	Tenant         string `json:"tenant"`
	Domain         string `json:"domain"`
	Network        string `json:"network"`
	SecurityGroups string `json:"securityGroups"`
	FloatingIPPool string `json:"floatingIpPool"`
	RouterID       string `json:"routerID"`
	SubnetID       string `json:"subnetID"`

	NetworkCreated       bool `json:"networkCreated"`
	SecurityGroupCreated bool `json:"securityGroupCreated"`
}

// ClusterHealthStatus stores health information of the components of a cluster.
type ClusterHealthStatus struct {
	Apiserver         bool `json:"apiserver"`
	Scheduler         bool `json:"scheduler"`
	Controller        bool `json:"controller"`
	MachineController bool `json:"machineController"`
	Etcd              bool `json:"etcd"`
}

// AllHealthy returns if all components are healthy
func (h *ClusterHealthStatus) AllHealthy() bool {
	return h.Etcd &&
		h.MachineController &&
		h.Controller &&
		h.Apiserver &&
		h.Scheduler
}

// MarshalJSON adds base64 json encoding to the Bytes type.
func (bs Bytes) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("\"%s\"", base64.StdEncoding.EncodeToString(bs))), nil
}

// UnmarshalJSON adds base64 json decoding to the Bytes type.
func (bs *Bytes) UnmarshalJSON(src []byte) error {
	if len(src) < 2 {
		return errors.New("base64 string expected")
	}
	if src[0] != '"' || src[len(src)-1] != '"' {
		return errors.New("\" quotations expected")
	}
	if len(src) == 2 {
		*bs = nil
		return nil
	}
	var err error
	*bs, err = base64.StdEncoding.DecodeString(string(src[1 : len(src)-1]))
	return err
}

// Base64 converts a Bytes instance to a base64 string.
func (bs Bytes) Base64() string {
	if []byte(bs) == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString([]byte(bs))
}

// NewBytes creates a Bytes instance from a base64 string, returning nil for an empty base64 string.
func NewBytes(b64 string) Bytes {
	if b64 == "" {
		return Bytes(nil)
	}
	bs, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(fmt.Sprintf("Invalid base64 string %q", b64))
	}
	return Bytes(bs)
}
