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
	"encoding/base64"
	"errors"
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// ClusterResourceName represents "Resource" defined in Kubernetes
	ClusterResourceName = "clusters"

	// ClusterKindName represents "Kind" defined in Kubernetes
	ClusterKindName = "Cluster"

	// AnnotationNameClusterAutoscalerEnabled is the name of the annotation that is being
	// used to determine if the cluster-autoscaler is enabled for this cluster. It is
	// enabled when this Annotation is set with any value
	AnnotationNameClusterAutoscalerEnabled = "kubermatic.io/cluster-autoscaler-enabled"

	// CredentialPrefix is the prefix used for the secrets containing cloud provider crednentials.
	CredentialPrefix = "credential"
)

const (
	WorkerNameLabelKey   = "worker-name"
	ProjectIDLabelKey    = "project-id"
	UpdatedByVPALabelKey = "updated-by-vpa"

	DefaultEtcdClusterSize = 3
	MaxEtcdClusterSize     = 9
)

// ProtectedClusterLabels is a set of labels that must not be set by users on clusters,
// as they are security relevant.
var ProtectedClusterLabels = sets.NewString(WorkerNameLabelKey, ProjectIDLabelKey)

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
	Version semver.Semver `json:"version"`
	// MasterVersion is Deprecated
	MasterVersion string `json:"masterVersion,omitempty"`

	// HumanReadableName is the cluster name provided by the user
	HumanReadableName string `json:"humanReadableName"`

	// ExposeStrategy is the approach we use to expose this cluster, either via NodePort
	// or via a dedicated LoadBalancer
	ExposeStrategy ExposeStrategy `json:"exposeStrategy"`

	// Pause tells that this cluster is currently not managed by the controller.
	// It indicates that the user needs to do some action to resolve the pause.
	Pause bool `json:"pause"`
	// PauseReason is the reason why the cluster is no being managed.
	PauseReason string `json:"pauseReason,omitempty"`

	// Optional component specific overrides
	ComponentsOverride ComponentSettings `json:"componentsOverride"`

	OIDC OIDCSettings `json:"oidc,omitempty"`

	// Feature flags
	// This unfortunately has to be a string map, because we use it in templating and that
	// can not cope with string types
	Features map[string]bool `json:"features,omitempty"`

	UpdateWindow *UpdateWindow `json:"updateWindow,omitempty"`

	// Openshift holds all openshift-specific settings
	Openshift *Openshift `json:"openshift,omitempty"`

	UsePodSecurityPolicyAdmissionPlugin bool     `json:"usePodSecurityPolicyAdmissionPlugin,omitempty"`
	UsePodNodeSelectorAdmissionPlugin   bool     `json:"usePodNodeSelectorAdmissionPlugin,omitempty"`
	AdmissionPlugins                    []string `json:"admissionPlugins,omitempty"`

	AuditLogging *AuditLoggingSettings `json:"auditLogging,omitempty"`

	// OPAIntegration is a preview feature that enables OPA integration with Kubermatic for the cluster.
	// Enabling it causes gatekeeper and its resources to be deployed on the user cluster.
	// By default it is disabled.
	OPAIntegration *OPAIntegrationSettings `json:"opaIntegration,omitempty"`
}

const (
	// ClusterFeatureExternalCloudProvider describes the external cloud provider feature. It is
	// only supported on a limited set of providers for a specific set of Kube versions. It must
	// not be set if its not supported.
	ClusterFeatureExternalCloudProvider = "externalCloudProvider"

	// ClusterFeatureRancherIntegration enables the rancher server integration feature.
	// It will deploy a Rancher Server Managegment plane on the seed cluster and import the user cluster into it.
	ClusterFeatureRancherIntegration = "rancherIntegration"

	// ClusterFeatureEtcdLauncher enables features related to the experimental etcd-launcher. This includes user-cluster
	// etcd scaling, automatic volume recovery and new backup/restore contorllers.
	ClusterFeatureEtcdLauncher = "etcdLauncher"
)

// ClusterConditionType is used to indicate the type of a cluster condition. For all condition
// types, the `true` value must indicate success. All condition types must be registered within
// the `AllClusterConditionTypes` variable.
type ClusterConditionType string

type UpdateWindow struct {
	Start  string `json:"start,omitempty"`
	Length string `json:"length,omitempty"`
}

const (
	// ClusterConditionSeedResourcesUpToDate indicates that all controllers have finished setting up the
	// resources for a user clusters that run inside the seed cluster, i.e. this ignores
	// the status of cloud provider resources for a given cluster.
	ClusterConditionSeedResourcesUpToDate ClusterConditionType = "SeedResourcesUpToDate"

	ClusterConditionClusterControllerReconcilingSuccess           ClusterConditionType = "ClusterControllerReconciledSuccessfully"
	ClusterConditionAddonControllerReconcilingSuccess             ClusterConditionType = "AddonControllerReconciledSuccessfully"
	ClusterConditionAddonInstallerControllerReconcilingSuccess    ClusterConditionType = "AddonInstallerControllerReconciledSuccessfully"
	ClusterConditionBackupControllerReconcilingSuccess            ClusterConditionType = "BackupControllerReconciledSuccessfully"
	ClusterConditionCloudControllerReconcilingSuccess             ClusterConditionType = "CloudControllerReconcilledSuccessfully"
	ClusterConditionComponentDefaulterReconcilingSuccess          ClusterConditionType = "ComponentDefaulterReconciledSuccessfully"
	ClusterConditionUpdateControllerReconcilingSuccess            ClusterConditionType = "UpdateControllerReconciledSuccessfully"
	ClusterConditionMonitoringControllerReconcilingSuccess        ClusterConditionType = "MonitoringControllerReconciledSuccessfully"
	ClusterConditionOpenshiftControllerReconcilingSuccess         ClusterConditionType = "OpenshiftControllerReconciledSuccessfully"
	ClusterConditionMachineDeploymentControllerReconcilingSuccess ClusterConditionType = "MachineDeploymentReconciledSuccessfully"
	ClusterConditionClusterInitialized                            ClusterConditionType = "ClusterInitialized"

	ClusterConditionRancherInitialized     ClusterConditionType = "RancherInitializedSuccessfully"
	ClusterConditionRancherClusterImported ClusterConditionType = "RancherClusterImportedSuccessfully"

	ClusterConditionEtcdClusterInitialized ClusterConditionType = "EtcdClusterInitialized"

	ReasonClusterUpdateSuccessful = "ClusterUpdateSuccessful"
	ReasonClusterUpdateInProgress = "ClusterUpdateInProgress"
)

var AllClusterConditionTypes = []ClusterConditionType{
	ClusterConditionSeedResourcesUpToDate,
	ClusterConditionClusterControllerReconcilingSuccess,
	ClusterConditionAddonControllerReconcilingSuccess,
	ClusterConditionBackupControllerReconcilingSuccess,
	ClusterConditionCloudControllerReconcilingSuccess,
	ClusterConditionComponentDefaulterReconcilingSuccess,
	ClusterConditionUpdateControllerReconcilingSuccess,
	ClusterConditionMonitoringControllerReconcilingSuccess,
	ClusterConditionOpenshiftControllerReconcilingSuccess,
}

type ClusterCondition struct {
	// Type of cluster condition.
	Type ClusterConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// KubermaticVersion current kubermatic version.
	KubermaticVersion string `json:"kubermatic_version"`
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
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

// ClusterStatus stores status information about a cluster.
type ClusterStatus struct {
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
	// ExtendedHealth exposes information about the current health state.
	// Extends standard health status for new states.
	ExtendedHealth ExtendedClusterHealth `json:"extendedHealth,omitempty"`
	// KubermaticVersion is the current kubermatic version in a cluster.
	KubermaticVersion string `json:"kubermatic_version"`
	// Deprecated
	RootCA *KeyCert `json:"rootCA,omitempty"`
	// Deprecated
	ApiserverCert *KeyCert `json:"apiserverCert,omitempty"`
	// Deprecated
	KubeletCert *KeyCert `json:"kubeletCert,omitempty"`
	// Deprecated
	ApiserverSSHKey *RSAKeys `json:"apiserverSshKey,omitempty"`
	// Deprecated
	ServiceAccountKey Bytes `json:"serviceAccountKey,omitempty"`
	// NamespaceName defines the namespace the control plane of this cluster is deployed in
	NamespaceName string `json:"namespaceName"`

	// UserName contains the name of the owner of this cluster
	UserName string `json:"userName,omitempty"`
	// UserEmail contains the email of the owner of this cluster
	UserEmail string `json:"userEmail"`

	// ErrorReason contains a error reason in case the controller encountered an error. Will be reset if the error was resolved
	ErrorReason *ClusterStatusError `json:"errorReason,omitempty"`
	// ErrorMessage contains a default error message in case the controller encountered an error. Will be reset if the error was resolved
	ErrorMessage *string `json:"errorMessage,omitempty"`

	// Conditions contains conditions the cluster is in, its primary use case is status signaling between controllers or between
	// controllers and the API
	Conditions []ClusterCondition `json:"conditions,omitempty"`

	// CloudMigrationRevision describes the latest version of the migration that has been done
	// It is used to avoid redundant and potentially costly migrations
	CloudMigrationRevision int `json:"cloudMigrationRevision"`

	// InheritedLabels are labels the cluster inherited from the project. They are read-only for users.
	InheritedLabels map[string]string `json:"inheritedLabels,omitempty"`
}

// HasConditionValue returns true if the cluster status has the given condition with the given status.
// It does not verify that the condition has been set by a certain Kubermatic version, it just checks
// the existence.
func (cs *ClusterStatus) HasConditionValue(conditionType ClusterConditionType, conditionStatus corev1.ConditionStatus) bool {
	for _, clusterCondition := range cs.Conditions {
		if clusterCondition.Type == conditionType {
			return clusterCondition.Status == conditionStatus
		}
	}

	return false
}

type ClusterStatusError string

const (
	InvalidConfigurationClusterError ClusterStatusError = "InvalidConfiguration"
	UnsupportedChangeClusterError    ClusterStatusError = "UnsupportedChange"
	ReconcileClusterError            ClusterStatusError = "ReconcileError"
)

type Openshift struct {
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
}

type OIDCSettings struct {
	IssuerURL     string `json:"issuerUrl,omitempty"`
	ClientID      string `json:"clientId,omitempty"`
	ClientSecret  string `json:"clientSecret,omitempty"`
	UsernameClaim string `json:"usernameClaim,omitempty"`
	GroupsClaim   string `json:"groupsClaim,omitempty"`
	RequiredClaim string `json:"requiredClaim,omitempty"`
	ExtraScopes   string `json:"extraScopes,omitempty"`
}

type AuditLoggingSettings struct {
	Enabled bool `json:"enabled,omitempty"`
}

type OPAIntegrationSettings struct {
	Enabled bool `json:"enabled,omitempty"`
}

type ComponentSettings struct {
	Apiserver         APIServerSettings       `json:"apiserver"`
	ControllerManager DeploymentSettings      `json:"controllerManager"`
	Scheduler         DeploymentSettings      `json:"scheduler"`
	Etcd              EtcdStatefulSetSettings `json:"etcd"`
	Prometheus        StatefulSetSettings     `json:"prometheus"`
}

type APIServerSettings struct {
	DeploymentSettings `json:",inline"`

	EndpointReconcilingDisabled *bool `json:"endpointReconcilingDisabled,omitempty"`
}

type DeploymentSettings struct {
	Replicas  *int32                       `json:"replicas,omitempty"`
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type StatefulSetSettings struct {
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type EtcdStatefulSetSettings struct {
	ClusterSize int                          `json:"clusterSize,omitempty"`
	Resources   *corev1.ResourceRequirements `json:"resources,omitempty"`
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

	// ProxyMode defines the kube-proxy mode (ipvs/iptables).
	// Defaults to ipvs.
	ProxyMode string `json:"proxyMode"`
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
	// Port is the port the API server listens on
	Port int32 `json:"port"`
	// ExternalName is the DNS name for this cluster
	ExternalName string `json:"externalName"`
	// InternalName is the seed cluster internal absolute DNS name to the API server
	InternalName string `json:"internalURL"`
	// AdminToken is the token for the kubeconfig, the user can download
	AdminToken string `json:"adminToken"`
	// IP is the external IP under which the apiserver is available
	IP string `json:"ip"`
	// OpenshiftConsoleCallBack is the callback address for the Openshift console
	OpenshiftConsoleCallBack string `json:"openshiftConsoleCallback,omitempty"`
}

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
	Packet       *PacketCloudSpec       `json:"packet,omitempty"`
	Hetzner      *HetznerCloudSpec      `json:"hetzner,omitempty"`
	VSphere      *VSphereCloudSpec      `json:"vsphere,omitempty"`
	GCP          *GCPCloudSpec          `json:"gcp,omitempty"`
	Kubevirt     *KubevirtCloudSpec     `json:"kubevirt,omitempty"`
	Alibaba      *AlibabaCloudSpec      `json:"alibaba,omitempty"`
	Anexia       *AnexiaCloudSpec       `json:"anexia,omitempty"`
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
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	Token string `json:"token,omitempty"` // Token is used to authenticate with the DigitalOcean API.
}

// HetznerCloudSpec specifies access data to hetzner cloud.
type HetznerCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	Token string `json:"token,omitempty"` // Token is used to authenticate with the Hetzner cloud API.
}

// AzureCloudSpec specifies access credentials to Azure cloud.
type AzureCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	TenantID       string `json:"tenantID,omitempty"`
	SubscriptionID string `json:"subscriptionID,omitempty"`
	ClientID       string `json:"clientID,omitempty"`
	ClientSecret   string `json:"clientSecret,omitempty"`

	ResourceGroup   string `json:"resourceGroup"`
	VNetName        string `json:"vnet"`
	SubnetName      string `json:"subnet"`
	RouteTableName  string `json:"routeTable"`
	SecurityGroup   string `json:"securityGroup"`
	AvailabilitySet string `json:"availabilitySet"`
}

// VSphereCredentials credentials represents a credential for accessing vSphere
type VSphereCredentials struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// VSphereCloudSpec specifies access data to VSphere cloud.
type VSphereCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	// Username is the vSphere user name.
	// +optional
	Username string `json:"username"`
	// Password is the vSphere user password.
	// +optional
	Password string `json:"password"`
	// VMNetName is the name of the vSphere network.
	VMNetName string `json:"vmNetName"`
	// Folder is the folder to be used to group the provisioned virtual
	// machines.
	// +optional
	Folder string `json:"folder"`
	// If both Datastore and DatastoreCluster are not specified the virtual
	// machines are stored in the `DefaultDatastore` specified for the
	// Datacenter.

	// Datastore to be used for storing virtual machines and as a default for
	// dynamic volume provisioning, it is mutually exclusive with
	// DatastoreCluster.
	// +optional
	Datastore string `json:"datastore,omitempty"`
	// DatastoreCluster to be used for storing virtual machines, it is mutually
	// exclusive with Datastore.
	// +optional
	DatastoreCluster string `json:"datastoreCluster,omitempty"`

	// This user will be used for everything except cloud provider functionality
	InfraManagementUser VSphereCredentials `json:"infraManagementUser"`
}

// BringYourOwnCloudSpec specifies access data for a bring your own cluster.
type BringYourOwnCloudSpec struct{}

// AWSCloudSpec specifies access data to Amazon Web Services.
type AWSCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	AccessKeyID     string `json:"accessKeyId,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	VPCID           string `json:"vpcId"`
	// The IAM role, the control plane will use. The control plane will perform an assume-role
	ControlPlaneRoleARN string `json:"roleARN"`
	RouteTableID        string `json:"routeTableId"`
	InstanceProfileName string `json:"instanceProfileName"`
	SecurityGroupID     string `json:"securityGroupID"`

	// DEPRECATED. Don't care for the role name. We only require the ControlPlaneRoleARN to be set so the control plane
	// can perform the assume-role.
	// We keep it for backwards compatibility (We use this name for cleanup purpose).
	RoleName string `json:"roleName,omitempty"`
}

// OpenstackCloudSpec specifies access data to an OpenStack cloud.
type OpenstackCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Tenant   string `json:"tenant,omitempty"`
	TenantID string `json:"tenantID,omitempty"`
	Domain   string `json:"domain,omitempty"`
	// Network holds the name of the internal network
	// When specified, all worker nodes will be attached to this network. If not specified, a network, subnet & router will be created
	//
	// Note that the network is internal if the "External" field is set to false
	Network        string `json:"network"`
	SecurityGroups string `json:"securityGroups"`
	// FloatingIPPool holds the name of the public network
	// The public network is reachable from the outside world
	// and should provide the pool of IP addresses to choose from.
	//
	// When specified, all worker nodes will receive a public ip from this floating ip pool
	//
	// Note that the network is external if the "External" field is set to true
	FloatingIPPool string `json:"floatingIpPool"`
	RouterID       string `json:"routerID"`
	SubnetID       string `json:"subnetID"`
}

// PacketCloudSpec specifies access data to a Packet cloud.
type PacketCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	APIKey       string `json:"apiKey,omitempty"`
	ProjectID    string `json:"projectID,omitempty"`
	BillingCycle string `json:"billingCycle"`
}

// GCPCloudSpec specifies access data to GCP.
type GCPCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	ServiceAccount string `json:"serviceAccount,omitempty"`
	Network        string `json:"network"`
	Subnetwork     string `json:"subnetwork"`
}

// KubevirtCloudSpec specifies the access data to Kubevirt.
type KubevirtCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	Kubeconfig string `json:"kubeconfig,omitempty"`
}

// AlibabaCloudSpec specifies the access data to Alibaba.
type AlibabaCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	AccessKeyID     string `json:"accessKeyId,omitempty"`
	AccessKeySecret string `json:"accessKeySecret,omitempty"`
}

// AnexiaCloudSpec specifies the access data to Anexia.
type AnexiaCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	Token string `json:"token,omitempty"`
}

type HealthStatus int

const (
	HealthStatusDown         HealthStatus = iota
	HealthStatusUp           HealthStatus = iota
	HealthStatusProvisioning HealthStatus = iota
)

// ExtendedClusterHealth stores health information of a cluster.
type ExtendedClusterHealth struct {
	Apiserver                    HealthStatus `json:"apiserver"`
	Scheduler                    HealthStatus `json:"scheduler"`
	Controller                   HealthStatus `json:"controller"`
	MachineController            HealthStatus `json:"machineController"`
	Etcd                         HealthStatus `json:"etcd"`
	OpenVPN                      HealthStatus `json:"openvpn"`
	CloudProviderInfrastructure  HealthStatus `json:"cloudProviderInfrastructure"`
	UserClusterControllerManager HealthStatus `json:"userClusterControllerManager"`
}

// AllHealthy returns if all components are healthy
func (h *ExtendedClusterHealth) AllHealthy() bool {
	return h.Etcd == HealthStatusUp &&
		h.MachineController == HealthStatusUp &&
		h.Controller == HealthStatusUp &&
		h.Apiserver == HealthStatusUp &&
		h.Scheduler == HealthStatusUp &&
		h.CloudProviderInfrastructure == HealthStatusUp &&
		h.UserClusterControllerManager == HealthStatusUp
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

func (cluster *Cluster) GetSecretName() string {
	if cluster.Spec.Cloud.AWS != nil {
		return fmt.Sprintf("%s-aws-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.Azure != nil {
		return fmt.Sprintf("%s-azure-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.Digitalocean != nil {
		return fmt.Sprintf("%s-digitalocean-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.GCP != nil {
		return fmt.Sprintf("%s-gcp-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.Hetzner != nil {
		return fmt.Sprintf("%s-hetzner-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.Openstack != nil {
		return fmt.Sprintf("%s-openstack-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.Packet != nil {
		return fmt.Sprintf("%s-packet-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.Kubevirt != nil {
		return fmt.Sprintf("%s-kubevirt-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.VSphere != nil {
		return fmt.Sprintf("%s-vsphere-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.Alibaba != nil {
		return fmt.Sprintf("%s-alibaba-%s", CredentialPrefix, cluster.Name)
	}
	if cluster.Spec.Cloud.Anexia != nil {
		return fmt.Sprintf("%s-anexia-%s", CredentialPrefix, cluster.Name)
	}
	return ""
}

func (cluster *Cluster) IsOpenshift() bool {
	return cluster.Annotations["kubermatic.io/openshift"] != ""
}

func (cluster *Cluster) IsKubernetes() bool {
	return !cluster.IsOpenshift()
}
