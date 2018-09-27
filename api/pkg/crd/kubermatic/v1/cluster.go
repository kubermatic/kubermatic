package v1

import (
	"encoding/base64"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ClusterResourceName represents "Resource" defined in Kubernetes
	ClusterResourceName = "clusters"

	// ClusterKindName represents "Kind" defined in Kubernetes
	ClusterKindName = "Cluster"
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

const (
	WorkerNameLabelKey = "worker-name"
	ProjectIDLabelKey  = "project-id"
)

//+genclient
//+genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the object representing a cluster.
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec    ClusterSpec    `json:"spec"`
	Address ClusterAddress `json:"address,omitempty"`
	Status  ClusterStatus  `json:"status,omitempty"`
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
	Cloud           CloudSpec                 `json:"cloud"`
	ClusterNetwork  ClusterNetworkingConfig   `json:"clusterNetwork"`
	MachineNetworks []MachineNetworkingConfig `json:"machineNetworks,omitempty"`

	// Version defines the wanted version of the control plane
	Version string `json:"version"`
	// MasterVersion is Deprecated
	MasterVersion string `json:"masterVersion"`

	// HumanReadableName is the cluster name provided by the user
	HumanReadableName string `json:"humanReadableName"`

	// Pause tells that this cluster is currently not managed by the controller.
	// It indicates that the user needs to do some action to resolve the pause.
	Pause bool `json:"pause"`
	// PauseReason is the reason why the cluster is no being managed.
	PauseReason string `json:"pauseReason,omitempty"`

	// Optional component specific overrides
	ComponentsOverride ComponentSettings `json:"componentsOverride"`
}

type ComponentSettings struct {
	Apiserver         DeploymentSettings `json:"apiserver"`
	ControllerManager DeploymentSettings `json:"controllerManager"`
	Scheduler         DeploymentSettings `json:"scheduler"`
	Etcd              EtcdSettings       `json:"etcd"`
}

type DeploymentSettings struct {
	Replicas  *int32                       `json:"replicas,omitempty"`
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type EtcdSettings struct {
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
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

// MachineNetworkingConfig specifies the networking parameters used for IPAM.
type MachineNetworkingConfig struct {
	CIDR       string   `json:"cidr"`
	Gateway    string   `json:"gateway"`
	DNSServers []string `json:"dnsServers"`
}

// NetworkRanges represents ranges of network addresses.
type NetworkRanges struct {
	CIDRBlocks []string `json:"cidrBlocks"`
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	// URL under which the Apiserver is available
	URL string `json:"url"`
	// ExternalName is the DNS name for this cluster
	ExternalName string `json:"externalName"`
	// AdminToken is the token for the kubeconfig, the user can download
	AdminToken string `json:"adminToken"`
	// IP is the external IP under which the apiserver is available
	IP string `json:"ip"`
}

// ClusterStatus stores status information about a cluster.
type ClusterStatus struct {
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
	// Phase is Deprecated.
	Phase ClusterPhase `json:"phase,omitempty"`
	// Health exposes information about the current health state of the individual control plane components
	Health ClusterHealth `json:"health,omitempty"`

	// Deprecated
	RootCA KeyCert `json:"rootCA"`
	// Deprecated
	ApiserverCert KeyCert `json:"apiserverCert"`
	// Deprecated
	KubeletCert KeyCert `json:"kubeletCert"`
	// Deprecated
	ApiserverSSHKey RSAKeys `json:"apiserverSshKey"`
	// Deprecated
	ServiceAccountKey Bytes `json:"serviceAccountKey"`
	// NamespaceName defines the namespace the control plane of this cluster is deployed in
	NamespaceName string `json:"namespaceName"`

	// UserName contains the name of the owner of this cluster
	UserName string `json:"userName"`
	// UserEmail contains the email of the owner of this cluster
	UserEmail string `json:"userEmail"`

	// ErrorReason contains a error reason in case the controller encountered an error. Will be reset if the error was resolved
	ErrorReason *ClusterStatusError `json:"errorReason,omitempty"`
	// ErrorMessage contains a defauled error message in case the controller encountered an error. Will be reset if the error was resolved
	ErrorMessage *string `json:"errorMessage,omitempty"`
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
	Azure        *AzureCloudSpec        `json:"azure,omitempty"`
	Openstack    *OpenstackCloudSpec    `json:"openstack,omitempty"`
	Hetzner      *HetznerCloudSpec      `json:"hetzner,omitempty"`
	VSphere      *VSphereCloudSpec      `json:"vsphere,omitempty"`
}

// UpdateCloudSpec is a helper method for updating the cloud spec
// since cloud credentials are removed from responses sometimes they will be empty
// for example when a client wants to revoke a token it will send the whole cluster object
// thus if credentials for specific cloud provider are empty rewrite from an existing object
func UpdateCloudSpec(updatedShared, existingShared CloudSpec) CloudSpec {
	updated := updatedShared.DeepCopy()

	if updated.Digitalocean != nil && len(updated.Digitalocean.Token) == 0 {
		updated.Digitalocean.Token = existingShared.Digitalocean.Token
	}
	if updated.AWS != nil {
		if len(updated.AWS.AccessKeyID) == 0 {
			updated.AWS.AccessKeyID = existingShared.AWS.AccessKeyID
		}
		if len(updated.AWS.SecretAccessKey) == 0 {
			updated.AWS.SecretAccessKey = existingShared.AWS.SecretAccessKey
		}
	}
	if updated.Azure != nil {
		if len(updated.Azure.ClientID) == 0 {
			updated.Azure.ClientID = existingShared.Azure.ClientID
		}
		if len(updated.Azure.ClientSecret) == 0 {
			updated.Azure.ClientSecret = existingShared.Azure.ClientSecret
		}
	}
	if updated.Openstack != nil {
		if len(updated.Openstack.Username) == 0 {
			updated.Openstack.Username = existingShared.Openstack.Username
		}
		if len(updated.Openstack.Password) == 0 {
			updated.Openstack.Password = existingShared.Openstack.Password
		}
	}
	if updated.Hetzner != nil && len(updated.Hetzner.Token) == 0 {
		updated.Hetzner.Token = existingShared.Hetzner.Token
	}
	if updated.VSphere != nil {
		if len(updated.VSphere.Username) == 0 {
			updated.VSphere.Username = existingShared.VSphere.Username
		}
		if len(updated.VSphere.Password) == 0 {
			updated.VSphere.Password = existingShared.VSphere.Password
		}
		if len(updated.VSphere.InfraManagementUser.Username) == 0 {
			updated.VSphere.InfraManagementUser.Username = existingShared.VSphere.InfraManagementUser.Username
		}
		if len(updated.VSphere.InfraManagementUser.Password) == 0 {
			updated.VSphere.InfraManagementUser.Password = existingShared.VSphere.InfraManagementUser.Password
		}
	}

	return *updated
}

// RemoveSensitiveDataFromCloudSpec remove credentials from cloud providers
func RemoveSensitiveDataFromCloudSpec(spec CloudSpec) CloudSpec {
	if spec.Digitalocean != nil {
		spec.Digitalocean.Token = ""
	}
	if spec.AWS != nil {
		spec.AWS.AccessKeyID = ""
		spec.AWS.SecretAccessKey = ""
	}
	if spec.Azure != nil {
		spec.Azure.ClientID = ""
		spec.Azure.ClientSecret = ""
	}
	if spec.Openstack != nil {
		spec.Openstack.Username = ""
		spec.Openstack.Password = ""
	}
	if spec.Hetzner != nil {
		spec.Hetzner.Token = ""
	}
	if spec.VSphere != nil {
		spec.VSphere.Username = ""
		spec.VSphere.Password = ""
		spec.VSphere.InfraManagementUser.Username = ""
		spec.VSphere.InfraManagementUser.Password = ""
	}

	return spec
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

// AzureCloudSpec specifies acceess credentials to Azure cloud.
type AzureCloudSpec struct {
	TenantID       string `json:"tenantID"`
	SubscriptionID string `json:"subscriptionID"`
	ClientID       string `json:"clientID"`
	ClientSecret   string `json:"clientSecret"`

	ResourceGroup   string `json:"resourceGroup"`
	VNetName        string `json:"vnet"`
	SubnetName      string `json:"subnet"`
	RouteTableName  string `json:"routeTable"`
	SecurityGroup   string `json:"securityGroup"`
	AvailabilitySet string `json:"availabilitySet"`
}

// VSphereCredentials credentials represents a credential for accessing vSphere
type VSphereCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// VSphereCloudSpec specifies access data to VSphere cloud.
type VSphereCloudSpec struct {
	Username  string `json:"username"`
	Password  string `json:"password"`
	VMNetName string `json:"vmNetName"`

	// This user will be used for everything except cloud provider functionality
	InfraManagementUser VSphereCredentials `json:"infraManagementUser"`
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
}

// ClusterHealthStatus stores health information of the components of a cluster.
type ClusterHealthStatus struct {
	Apiserver         bool `json:"apiserver"`
	Scheduler         bool `json:"scheduler"`
	Controller        bool `json:"controller"`
	MachineController bool `json:"machineController"`
	Etcd              bool `json:"etcd"`
	OpenVPN           bool `json:"openvpn"`
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
