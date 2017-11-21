package v1

import (
	"encoding/base64"
	"errors"
	"fmt"

	nodesetclient "github.com/kube-node/nodeset/pkg/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	cmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
	cmdv1 "k8s.io/client-go/tools/clientcmd/api/v1"
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

	// PendingClusterStatusPhase means that the cluster controller hasn't picked the cluster up.
	PendingClusterStatusPhase ClusterPhase = "Pending"

	// LaunchingClusterStatusPhase means that the cluster controller starts up the cluster.
	LaunchingClusterStatusPhase ClusterPhase = "Launching"

	// FailedClusterStatusPhase means that the cluster controller time out launching the cluster.
	FailedClusterStatusPhase ClusterPhase = "Failed"

	// RunningClusterStatusPhase means that the cluster is cluster is up and running.
	RunningClusterStatusPhase ClusterPhase = "Running"

	// DeletingClusterStatusPhase means that the cluster controller is deleting the cluster.
	DeletingClusterStatusPhase ClusterPhase = "Deleting"

	// UpdatingMasterClusterStatusPhase means that the cluster controller is updating the master components of the cluster.
	UpdatingMasterClusterStatusPhase ClusterPhase = "Updatingmaster"
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

const (
	WorkerNameLabelKey = "worker-name"
)

//+genclient
//+genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is the object representing a cluster.
// swagger:response KubeCluster
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
	Cloud *CloudSpec `json:"cloud"`

	HumanReadableName  string `json:"humanReadableName"` // HumanReadableName is the cluster name provided by the user
	MasterVersion      string `json:"masterVersion"`
	WorkerName         string `json:"workerName"` // WorkerName is a cluster used in development, compare --worker-name flag.
	SeedDatacenterName string `json:"seedDatacenterName"`
}

// ClusterAddress stores access and address information of a cluster.
type ClusterAddress struct {
	URL          string `json:"url"`
	ExternalName string `json:"externalName"`
	ExternalPort int    `json:"externalPort"`
	KubeletToken string `json:"kubeletToken"`
	AdminToken   string `json:"adminToken"`
}

// ClusterStatus stores status information about a cluster.
type ClusterStatus struct {
	LastTransitionTime        metav1.Time       `json:"lastTransitionTime"`
	Phase                     ClusterPhase      `json:"phase,omitempty"`
	Health                    *ClusterHealth    `json:"health,omitempty"`
	LastDeployedMasterVersion string            `json:"lastDeployedMasterVersion"`
	MasterUpdatePhase         MasterUpdatePhase `json:"masterUpdatePhase"`

	RootCA            KeyCert `json:"rootCA"`
	ApiserverCert     KeyCert `json:"apiserverCert"`
	KubeletCert       KeyCert `json:"kubeletCert"`
	ApiserverSSHKey   RSAKeys `json:"apiserverSshKey"`
	ServiceAccountKey Bytes   `json:"serviceAccountKey"`
	Seed              string  `json:"seed,omitempty"`
	NamespaceName     string  `json:"namespaceName"`

	// Conditions Represents the latest available observations of the cluster's current state.
	Conditions []ClusterCondition `json:"conditions,omitempty"`

	UserName  string `json:"userName"`
	UserEmail string `json:"userEmail"`
}

type ClusterConditionType string

const (
	ClusterNodeCleanuoFailure ClusterConditionType = "NodeCleanupFailure"
)

func getClusterCondition(status *ClusterStatus, t ClusterConditionType) (int, *ClusterCondition) {
	for i, c := range status.Conditions {
		if t == c.Type {
			return i, &c
		}
	}
	return -1, nil
}

func newClusterCondition(condType ClusterConditionType, status corev1.ConditionStatus, reason, message string) *ClusterCondition {
	return &ClusterCondition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	}
}

func (cs *ClusterStatus) ClearCondition(t ClusterConditionType) {
	pos, _ := getClusterCondition(cs, t)
	if pos == -1 {
		return
	}
	cs.Conditions = append(cs.Conditions[:pos], cs.Conditions[pos+1:]...)
}

func (cs *ClusterStatus) setClusterCondition(c ClusterCondition) {
	pos, cp := getClusterCondition(cs, c.Type)
	if cp != nil &&
		cp.Status == c.Status && cp.Reason == c.Reason && cp.Message == c.Message {
		return
	}

	if cp != nil {
		cs.Conditions[pos] = c
	} else {
		cs.Conditions = append(cs.Conditions, c)
	}
}

// NodeSetCondition describes the state of a NodeSet at a certain point.
type ClusterCondition struct {
	// Type of Cluster condition.
	Type ClusterConditionType `json:"type"`

	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`

	// The last time the condition transitioned from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`

	// The reason for the condition's last transition.
	// +optional
	Reason string `json:"reason,omitempty"`

	// A human readable message indicating details about the transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// CloudSpec mutually stores access data to a cloud provider.
type CloudSpec struct {
	// DatacenterName where the users 'cloud' lives in.
	DatacenterName string `json:"dc"`

	Fake         *FakeCloudSpec         `json:"fake,omitempty"`
	Digitalocean *DigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	BringYourOwn *BringYourOwnCloudSpec `json:"bringyourown,omitempty"`
	AWS          *AWSCloudSpec          `json:"aws,omitempty"`
	BareMetal    *BareMetalCloudSpec    `json:"baremetal,omitempty"`
	Openstack    *OpenstackCloudSpec    `json:"openstack,omitempty"`
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

// DigitaloceanCloudSpec specifies access data to digital ocean.
type DigitaloceanCloudSpec struct {
	Token string `json:"token"` // Token is used to authenticate with the DigitalOcean API.
}

// BringYourOwnCloudSpec specifies access data for a bring your own cluster.
type BringYourOwnCloudSpec struct {
	PrivateIntf   string  `json:"privateInterface"`
	ClientKeyCert KeyCert `json:"clientKeyCert"`
}

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

	AvailabilityZone string `json:"availabilityZone"`
}

// BareMetalCloudSpec specifies access to a bare metal datacenter
type BareMetalCloudSpec struct {
	Name string `json:"name"`
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
	Apiserver      bool `json:"apiserver"`
	Scheduler      bool `json:"scheduler"`
	Controller     bool `json:"controller"`
	NodeController bool `json:"nodeController"`
	Etcd           bool `json:"etcd"`
}

// AllHealthy returns if all components are healthy
func (h *ClusterHealthStatus) AllHealthy() bool {
	return h.Etcd &&
		h.NodeController &&
		h.Controller &&
		h.Apiserver &&
		h.Scheduler
}

// GetKubeconfig returns a kubeconfig to connect to the cluster
func (c *Cluster) GetKubeconfig() *cmdv1.Config {
	return &cmdv1.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		CurrentContext: c.ObjectMeta.Name,
		Clusters: []cmdv1.NamedCluster{{
			Name: c.ObjectMeta.Name,
			Cluster: cmdv1.Cluster{
				Server: c.Address.URL,
				CertificateAuthorityData: c.Status.RootCA.Cert,
			},
		}},
		Contexts: []cmdv1.NamedContext{{
			Name: c.ObjectMeta.Name,
			Context: cmdv1.Context{
				Cluster:  c.ObjectMeta.Name,
				AuthInfo: c.ObjectMeta.Name,
			},
		}},
		AuthInfos: []cmdv1.NamedAuthInfo{{
			Name: c.ObjectMeta.Name,
			AuthInfo: cmdv1.AuthInfo{
				Token: c.Address.AdminToken,
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

	client := kubernetes.NewForConfigOrDie(ccfg)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// GetNodesetClient returns a client interact with nodeset resources
func (c *Cluster) GetNodesetClient() (nodesetclient.Interface, error) {
	cfg, err := c.getClientConfig()
	if err != nil {
		return nil, err
	}

	ccfg, err := cfg.ClientConfig()
	if err != nil {
		return nil, err
	}

	client, err := nodesetclient.NewForConfig(ccfg)
	if err != nil {
		return nil, err
	}

	return client, nil
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
