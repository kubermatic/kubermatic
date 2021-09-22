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
	"k8s.io/apimachinery/pkg/api/resource"
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

	// ForceRestartAnnotation is key of the annotation used to restart machine deployments.
	ForceRestartAnnotation = "forceRestart"
)

const (
	CCMMigrationNeededAnnotation = "ccm-migration.k8c.io/migration-needed"
	CSIMigrationNeededAnnotation = "csi-migration.k8c.io/migration-needed"
)

const (
	WorkerNameLabelKey   = "worker-name"
	ProjectIDLabelKey    = "project-id"
	UpdatedByVPALabelKey = "updated-by-vpa"

	DefaultEtcdClusterSize = 3
	MinEtcdClusterSize     = 3
	MaxEtcdClusterSize     = 9
)

type LBSKU string

const (
	AzureStandardLBSKU = LBSKU("standard")
	AzureBasicLBSKU    = LBSKU("basic")
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

	UsePodSecurityPolicyAdmissionPlugin bool `json:"usePodSecurityPolicyAdmissionPlugin,omitempty"`
	UsePodNodeSelectorAdmissionPlugin   bool `json:"usePodNodeSelectorAdmissionPlugin,omitempty"`

	// EnableUserSSHKeyAgent control whether the UserSSHKeyAgent will be deployed in the user cluster or not.
	// If it was enabled, the agent will be deployed and used to sync the user ssh keys, that the user attach
	// to the created cluster. If the agent was disabled, it won't be deployed in the user cluster, thus after
	// the cluster creation any attached ssh keys won't be synced to the worker nodes. Once the agent is enabled/disabled
	// it cannot be changed after the cluster is being created.
	EnableUserSSHKeyAgent *bool `json:"enableUserSSHKeyAgent,omitempty"`

	// PodNodeSelectorAdmissionPluginConfig provides the configuration for the PodNodeSelector.
	// It's used by the backend to create a configuration file for this plugin.
	// The key:value from the map is converted to the namespace:<node-selectors-labels> in the file.
	// The format in a file:
	// podNodeSelectorPluginConfig:
	//  clusterDefaultNodeSelector: <node-selectors-labels>
	//  namespace1: <node-selectors-labels>
	//  namespace2: <node-selectors-labels>
	PodNodeSelectorAdmissionPluginConfig map[string]string `json:"podNodeSelectorAdmissionPluginConfig,omitempty"`
	AdmissionPlugins                     []string          `json:"admissionPlugins,omitempty"`

	AuditLogging *AuditLoggingSettings `json:"auditLogging,omitempty"`

	// OPAIntegration is a preview feature that enables OPA integration with Kubermatic for the cluster.
	// Enabling it causes gatekeeper and its resources to be deployed on the user cluster.
	// By default it is disabled.
	OPAIntegration *OPAIntegrationSettings `json:"opaIntegration,omitempty"`

	// ServiceAccount contains service account related settings for the kube-apiserver of user cluster.
	ServiceAccount *ServiceAccountSettings `json:"serviceAccount,omitempty"`

	// MLA contains monitoring, logging and alerting related settings for the user cluster.
	MLA *MLASettings `json:"mla,omitempty"`

	// ContainerRuntime to use, i.e. Docker or containerd. By default containerd will be used.
	ContainerRuntime string `json:"containerRuntime,omitempty"`

	// CNIPlugin contains the spec of the CNI plugin to be installed in the cluster.
	CNIPlugin *CNIPluginSettings `json:"cniPlugin,omitempty"`
}

// CNIPluginSettings contains the spec of the CNI plugin used by the Cluster.
type CNIPluginSettings struct {
	Type    CNIPluginType `json:"type"`
	Version string        `json:"version"`
}

// CNIPluginType define the type of CNI plugin installed. e.g. Canal
type CNIPluginType string

func (c CNIPluginType) String() string {
	return string(c)
}

const (
	// CNIPluginTypeCanal corresponds to Canal CNI plugin (i.e. Flannel +
	// Calico for policy enforcement).
	CNIPluginTypeCanal CNIPluginType = "canal"

	// CNIPluginTypeCilium corresponds to Cilium CNI plugin
	CNIPluginTypeCilium CNIPluginType = "cilium"
)

const (
	// ClusterFeatureExternalCloudProvider describes the external cloud provider feature. It is
	// only supported on a limited set of providers for a specific set of Kube versions. It must
	// not be set if its not supported.
	ClusterFeatureExternalCloudProvider = "externalCloudProvider"

	// ClusterFeatureCCMClusterName sets the cluster-name flag on the external CCM deployment.
	// The cluster-name flag is often used for naming cloud resources, such as load balancers.
	ClusterFeatureCCMClusterName = "ccmClusterName"

	// ClusterFeatureRancherIntegration enables the rancher server integration feature.
	// It will deploy a Rancher Server Managegment plane on the seed cluster and import the user cluster into it.
	ClusterFeatureRancherIntegration = "rancherIntegration"

	// ClusterFeatureEtcdLauncher enables features related to the experimental etcd-launcher. This includes user-cluster
	// etcd scaling, automatic volume recovery and new backup/restore contorllers.
	ClusterFeatureEtcdLauncher = "etcdLauncher"

	// ApiserverNetworkPolicy enables the deployment of network policies that
	// restrict the egress traffic from Apiserver pods.
	ApiserverNetworkPolicy = "apiserverNetworkPolicy"
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
	ClusterConditionUpdateControllerReconcilingSuccess            ClusterConditionType = "UpdateControllerReconciledSuccessfully"
	ClusterConditionMonitoringControllerReconcilingSuccess        ClusterConditionType = "MonitoringControllerReconciledSuccessfully"
	ClusterConditionMachineDeploymentControllerReconcilingSuccess ClusterConditionType = "MachineDeploymentReconciledSuccessfully"
	ClusterConditionMLAControllerReconcilingSuccess               ClusterConditionType = "MLAControllerReconciledSuccessfully"
	ClusterConditionClusterInitialized                            ClusterConditionType = "ClusterInitialized"

	ClusterConditionRancherInitialized     ClusterConditionType = "RancherInitializedSuccessfully"
	ClusterConditionRancherClusterImported ClusterConditionType = "RancherClusterImportedSuccessfully"

	ClusterConditionEtcdClusterInitialized ClusterConditionType = "EtcdClusterInitialized"

	// ClusterConditionNone is a special value indicating that no cluster condition should be set
	ClusterConditionNone ClusterConditionType = ""
	// This condition is met when a CSI migration is ongoing and the CSI
	// migration feature gates are activated on the Kubelets of all the nodes.
	// When this condition is `true` CSIMigration{provider}Complete can be
	// enabled.
	ClusterConditionCSIKubeletMigrationCompleted ClusterConditionType = "CSIKubeletMigrationCompleted"

	ReasonClusterUpdateSuccessful             = "ClusterUpdateSuccessful"
	ReasonClusterUpdateInProgress             = "ClusterUpdateInProgress"
	ReasonClusterCSIKubeletMigrationCompleted = "CSIKubeletMigrationSuccess"
	ReasonClusterCCMMigrationInProgress       = "CSIKubeletMigrationInProgress"
)

var AllClusterConditionTypes = []ClusterConditionType{
	ClusterConditionSeedResourcesUpToDate,
	ClusterConditionClusterControllerReconcilingSuccess,
	ClusterConditionAddonControllerReconcilingSuccess,
	ClusterConditionBackupControllerReconcilingSuccess,
	ClusterConditionCloudControllerReconcilingSuccess,
	ClusterConditionUpdateControllerReconcilingSuccess,
	ClusterConditionMonitoringControllerReconcilingSuccess,
}

type ClusterCondition struct {
	// Type of cluster condition.
	Type ClusterConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// KubermaticVersion current kubermatic version.
	KubermaticVersion string `json:"kubermaticVersion"`
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
	KubermaticVersion string `json:"kubermaticVersion"`
	// Deprecated
	RootCA *KeyCert `json:"rootCA,omitempty"` //nolint:tagliatelle
	// Deprecated
	ApiserverCert *KeyCert `json:"apiserverCert,omitempty"`
	// Deprecated
	KubeletCert *KeyCert `json:"kubeletCert,omitempty"`
	// Deprecated
	ApiserverSSHKey *RSAKeys `json:"apiserverSSHKey,omitempty"`
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

type OIDCSettings struct {
	IssuerURL     string `json:"issuerURL,omitempty"`
	ClientID      string `json:"clientID,omitempty"`
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
	// Enabled is the flag for enabling OPA integration
	Enabled bool `json:"enabled,omitempty"`
	// WebhookTimeout is the timeout that is set for the gatekeeper validating webhook admission review calls.
	// By default 10 seconds.
	WebhookTimeoutSeconds *int32 `json:"webhookTimeoutSeconds,omitempty"`
	// Enable mutation
	ExperimentalEnableMutation bool `json:"experimentalEnableMutation,omitempty"`
}

type ServiceAccountSettings struct {
	TokenVolumeProjectionEnabled bool `json:"tokenVolumeProjectionEnabled,omitempty"`
	// Issuer is the identifier of the service account token issuer
	// If this is not specified, it will be set to the URL of apiserver by default
	Issuer string `json:"issuer,omitempty"`
	// APIAudiences are the Identifiers of the API
	// If this is not specified, it will be set to a single element list containing the issuer URL
	APIAudiences []string `json:"apiAudiences,omitempty"`
}

type MLASettings struct {
	// MonitoringEnabled is the flag for enabling monitoring in user cluster.
	MonitoringEnabled bool `json:"monitoringEnabled,omitempty"`
	// LoggingEnabled is the flag for enabling logging in user cluster.
	LoggingEnabled bool `json:"loggingEnabled,omitempty"`
	// MonitoringResources is the resource requirements for user cluster prometheus.
	MonitoringResources *corev1.ResourceRequirements `json:"monitoringResources,omitempty"`
	// LoggingResources is the resource requirements for user cluster promtail.
	LoggingResources *corev1.ResourceRequirements `json:"loggingResources,omitempty"`
}

type ComponentSettings struct {
	Apiserver         APIServerSettings       `json:"apiserver"`
	ControllerManager ControllerSettings      `json:"controllerManager"`
	Scheduler         ControllerSettings      `json:"scheduler"`
	Etcd              EtcdStatefulSetSettings `json:"etcd"`
	Prometheus        StatefulSetSettings     `json:"prometheus"`
}

type APIServerSettings struct {
	DeploymentSettings `json:",inline"`

	EndpointReconcilingDisabled *bool  `json:"endpointReconcilingDisabled,omitempty"`
	NodePortRange               string `json:"nodePortRange,omitempty"`
}

type ControllerSettings struct {
	DeploymentSettings     `json:",inline"`
	LeaderElectionSettings `json:"leaderElection,omitempty"`
}

type DeploymentSettings struct {
	Replicas    *int32                       `json:"replicas,omitempty"`
	Resources   *corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations []corev1.Toleration          `json:"tolerations,omitempty"`
}

type StatefulSetSettings struct {
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

type EtcdStatefulSetSettings struct {
	ClusterSize  *int32                       `json:"clusterSize,omitempty"`
	StorageClass string                       `json:"storageClass,omitempty"`
	DiskSize     *resource.Quantity           `json:"diskSize,omitempty"`
	Resources    *corev1.ResourceRequirements `json:"resources,omitempty"`
	Tolerations  []corev1.Toleration          `json:"tolerations,omitempty"`
}

type LeaderElectionSettings struct {
	// LeaseDurationSeconds is the duration in seconds that non-leader candidates
	// will wait to force acquire leadership. This is measured against time of
	// last observed ack.
	// +optional
	LeaseDurationSeconds *int32 `json:"leaseDurationSeconds,omitempty"`
	// RenewDeadlineSeconds is the duration in seconds that the acting controlplane
	// will retry refreshing leadership before giving up.
	// +optional
	RenewDeadlineSeconds *int32 `json:"renewDeadlineSeconds,omitempty"`
	// RetryPeriodSeconds is the duration in seconds the LeaderElector clients
	// should wait between tries of actions.
	// +optional
	RetryPeriodSeconds *int32 `json:"retryPeriodSeconds,omitempty"`
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

	// IPVS defines kube-proxy ipvs configuration options
	IPVS *IPVSConfiguration `json:"ipvs,omitempty"`

	// NodeLocalDNSCacheEnabled controls whether the NodeLocal DNS Cache feature is enabled.
	// Defaults to true.
	NodeLocalDNSCacheEnabled *bool `json:"nodeLocalDNSCacheEnabled,omitempty"`

	// KonnectivityEnabled enables konnectivity for controlplane to node network communication.
	KonnectivityEnabled bool `json:"konnectivityEnabled,omitempty"`
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
}

// IPVSConfiguration contains ipvs-related configuration details for kube-proxy.
type IPVSConfiguration struct {
	// StrictArp configure arp_ignore and arp_announce to avoid answering ARP queries from kube-ipvs0 interface.
	// defaults to true.
	StrictArp *bool `json:"strictArp,omitempty"`
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

	// Token is used to authenticate with the Hetzner cloud API.
	Token string `json:"token,omitempty"`
	// Network is the pre-existing Hetzner network in which the machines are running.
	// While machines can be in multiple networks, a single one must be chosen for the
	// HCloud CCM to work.
	// If this is empty, the network configured on the datacenter will be used.
	Network string `json:"network,omitempty"`
}

// AzureCloudSpec specifies access credentials to Azure cloud.
type AzureCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	TenantID       string `json:"tenantID,omitempty"`
	SubscriptionID string `json:"subscriptionID,omitempty"`
	ClientID       string `json:"clientID,omitempty"`
	ClientSecret   string `json:"clientSecret,omitempty"`

	ResourceGroup         string `json:"resourceGroup"`
	VNetResourceGroup     string `json:"vnetResourceGroup"`
	VNetName              string `json:"vnet"`
	SubnetName            string `json:"subnet"`
	RouteTableName        string `json:"routeTable"`
	SecurityGroup         string `json:"securityGroup"`
	AssignAvailabilitySet *bool  `json:"assignAvailabilitySet"`
	AvailabilitySet       string `json:"availabilitySet"`
	// LoadBalancerSKU sets the LB type that will be used for the Azure cluster, possible values are "basic" and "standard", if empty, "basic" will be used
	LoadBalancerSKU LBSKU `json:"loadBalancerSKU"` //nolint:tagliatelle
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

	// StoragePolicy to be used for storage provisioning
	StoragePolicy string `json:"storagePolicy"`

	// ResourcePool is used to manage resources such as cpu and memory for vSphere virtual machines. The resource pool
	// should be defined on vSphere cluster level.
	// +optional
	ResourcePool string `json:"resourcePool,omitempty"`

	// This user will be used for everything except cloud provider functionality
	InfraManagementUser VSphereCredentials `json:"infraManagementUser"`
}

// BringYourOwnCloudSpec specifies access data for a bring your own cluster.
type BringYourOwnCloudSpec struct{}

// AWSCloudSpec specifies access data to Amazon Web Services.
type AWSCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`

	AccessKeyID     string `json:"accessKeyID,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	VPCID           string `json:"vpcID"`
	// The IAM role, the control plane will use. The control plane will perform an assume-role
	ControlPlaneRoleARN string `json:"roleARN"` //nolint:tagliatelle
	RouteTableID        string `json:"routeTableID"`
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

	Username                    string `json:"username,omitempty"`
	Password                    string `json:"password,omitempty"`
	Tenant                      string `json:"tenant,omitempty"`
	TenantID                    string `json:"tenantID,omitempty"`
	Domain                      string `json:"domain,omitempty"`
	ApplicationCredentialID     string `json:"applicationCredentialID,omitempty"`
	ApplicationCredentialSecret string `json:"applicationCredentialSecret,omitempty"`
	UseToken                    bool   `json:"useToken,omitempty"`
	// Used internally during cluster creation
	Token string `json:"token,omitempty"`

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
	FloatingIPPool string `json:"floatingIPPool"`
	RouterID       string `json:"routerID"`
	SubnetID       string `json:"subnetID"`
	// Whether or not to use Octavia for LoadBalancer type of Service
	// implementation instead of using Neutron-LBaaS.
	// Attention:Openstack CCM use Octavia as default load balancer
	// implementation since v1.17.0
	//
	// Takes precedence over the 'use_octavia' flag provided at datacenter
	// level if both are specified.
	// +optional
	UseOctavia *bool `json:"useOctavia,omitempty"`
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

	AccessKeyID     string `json:"accessKeyID,omitempty"`
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
	GatekeeperController         HealthStatus `json:"gatekeeperController,omitempty"`
	GatekeeperAudit              HealthStatus `json:"gatekeeperAudit,omitempty"`
}

// AllHealthy returns if all components are healthy. Gatekeeper components not included as they are optional and not
// crucial for cluster functioning
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

func (cluster *Cluster) GetUserClusterMLAResourceRequirements() map[string]*corev1.ResourceRequirements {
	if cluster.Spec.MLA == nil {
		return nil
	}
	return map[string]*corev1.ResourceRequirements{
		"monitoring": cluster.Spec.MLA.MonitoringResources,
		"logging":    cluster.Spec.MLA.LoggingResources,
	}
}
