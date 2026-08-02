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
	"fmt"

	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/machine-controller/sdk/providerconfig"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ExternalClusterResourceName represents "Resource" defined in Kubernetes.
	ExternalClusterResourceName = "externalclusters"

	// ExternalClusterKind represents "Kind" defined in Kubernetes.
	ExternalClusterKind = "ExternalCluster"

	// ExternalCluster Kubeconfig secret prefix.
	ExternalClusterKubeconfigPrefix = "kubeconfig-external-cluster"

	// KubeOneNamespacePrefix is the kubeone namespace prefix.
	KubeOneNamespacePrefix = "kubeone"

	// don't change this as these prefixes are used for rbac generation.
	// KubeOne ssh secret prefixes.
	KubeOneSSHSecretPrefix = "ssh-kubeone-external-cluster"

	// KubeOne manifest secret prefixes.
	KubeOneManifestSecretPrefix = "manifest-kubeone-external-cluster"
)

// +kubebuilder:validation:Enum=aks;bringyourown;eks;gke;kubeone

// ExternalClusterProvider is the identifier for the cloud provider that hosts
// the external cluster control plane.
type ExternalClusterProvider string

const (
	ExternalClusterAKSProvider          ExternalClusterProvider = "aks"
	ExternalClusterBringYourOwnProvider ExternalClusterProvider = "bringyourown"
	ExternalClusterEKSProvider          ExternalClusterProvider = "eks"
	ExternalClusterGKEProvider          ExternalClusterProvider = "gke"
	ExternalClusterKubeOneProvider      ExternalClusterProvider = "kubeone"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".spec.humanReadableName",name="HumanReadableName",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.cloudSpec.providerName",name="Provider",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.pause",name="Paused",type="boolean"
// +kubebuilder:printcolumn:JSONPath=".status.condition.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// ExternalCluster is the object representing an external kubernetes cluster.
type ExternalCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec describes the desired cluster state.
	Spec ExternalClusterSpec `json:"spec"`

	// Status contains reconciliation information for the cluster.
	Status ExternalClusterStatus `json:"status,omitempty"`
}

// ExternalClusterStatus denotes status information about an ExternalCluster.
type ExternalClusterStatus struct {
	// Conditions contains conditions an externalcluster is in, its primary use case is status signaling for controller
	Condition ExternalClusterCondition `json:"condition,omitempty"`
}

type ExternalClusterCondition struct {
	Phase ExternalClusterPhase `json:"phase"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`
}

type ExternalClusterKubeOneCloudSpec struct {
	// The name of the cloud provider used, one of
	// "aws", "azure", "digitalocean", "gcp",
	// "hetzner", "nutanix", "openstack", "vsphere" KubeOne natively-supported providers
	ProviderName string `json:"providerName"`

	// The cloud provider region in which the cluster resides.
	// This field is used only to display information.
	Region string `json:"region,omitempty"`

	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`
	SSHReference         *providerconfig.GlobalSecretKeySelector `json:"sshReference,omitempty"`
	ManifestReference    *providerconfig.GlobalSecretKeySelector `json:"manifestReference,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ExternalClusterList specifies a list of external kubernetes clusters.
type ExternalClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	// Items holds the list of the External Kubernetes cluster.
	Items []ExternalCluster `json:"items"`
}

// ExternalClusterSpec specifies the data for a new external kubernetes cluster.
type ExternalClusterSpec struct {
	// HumanReadableName is the cluster name provided by the user
	HumanReadableName string `json:"humanReadableName"`

	// Reference to cluster Kubeconfig
	KubeconfigReference *providerconfig.GlobalSecretKeySelector `json:"kubeconfigReference,omitempty"`

	// Defines the wanted version of the control plane.
	Version semver.Semver `json:"version"`

	// CloudSpec contains provider specific fields
	CloudSpec ExternalClusterCloudSpec `json:"cloudSpec"`

	// ClusterNetwork contains the different networking parameters for an external cluster.
	ClusterNetwork ExternalClusterNetworkingConfig `json:"clusterNetwork,omitempty"`

	// ContainerRuntime to use, i.e. `docker` or `containerd`.
	ContainerRuntime string `json:"containerRuntime,omitempty"`

	// If this is set to true, the cluster will not be reconciled by KKP.
	// This indicates that the user needs to do some action to resolve the pause.
	Pause bool `json:"pause"`

	// PauseReason is the reason why the cluster is not being managed. This field is for informational
	// purpose only and can be set by a user or a controller to communicate the reason for pausing the cluster.
	PauseReason string `json:"pauseReason,omitempty"`
}

// ExternalClusterNetworkingConfig specifies the different networking
// parameters for an external cluster.
type ExternalClusterNetworkingConfig struct {
	// The network ranges from which service VIPs are allocated.
	// It can contain one IPv4 and/or one IPv6 CIDR.
	// If both address families are specified, the first one defines the primary address family.
	Services ExternalClusterNetworkRanges `json:"services,omitempty"`

	// The network ranges from which POD networks are allocated.
	// It can contain one IPv4 and/or one IPv6 CIDR.
	// If both address families are specified, the first one defines the primary address family.
	Pods ExternalClusterNetworkRanges `json:"pods,omitempty"`
}

// ExternalClusterNetworkRanges represents ranges of network addresses.
type ExternalClusterNetworkRanges struct {
	CIDRBlocks []string `json:"cidrBlocks,omitempty"`
}

// ExternalClusterCloudSpec mutually stores access data to a cloud provider.
type ExternalClusterCloudSpec struct {
	ProviderName ExternalClusterProvider               `json:"providerName"`
	GKE          *ExternalClusterGKECloudSpec          `json:"gke,omitempty"`
	EKS          *ExternalClusterEKSCloudSpec          `json:"eks,omitempty"`
	AKS          *ExternalClusterAKSCloudSpec          `json:"aks,omitempty"`
	KubeOne      *ExternalClusterKubeOneCloudSpec      `json:"kubeone,omitempty"`
	BringYourOwn *ExternalClusterBringYourOwnCloudSpec `json:"bringyourown,omitempty"`
}

type ExternalClusterPhase string

const (
	// ExternalClusterPhaseProvisioning status indicates the cluster is being imported.
	ExternalClusterPhaseProvisioning ExternalClusterPhase = "Provisioning"

	// ExternalClusterPhaseRunning status indicates the cluster is fully usable.
	ExternalClusterPhaseRunning ExternalClusterPhase = "Running"

	ExternalClusterPhaseStarting ExternalClusterPhase = "Starting"

	ExternalClusterPhaseStopping ExternalClusterPhase = "Stopping"

	ExternalClusterPhaseStopped ExternalClusterPhase = "Stopped"

	ExternalClusterPhaseWarning ExternalClusterPhase = "Warning"

	// ExternalClusterPhaseReconciling status indicates that some work is actively being done on the cluster, such as upgrading the master or
	// node software. Details can be found in the `StatusMessage` field.
	ExternalClusterPhaseReconciling ExternalClusterPhase = "Reconciling"

	KubeOnePhaseReconcilingUpgrade ExternalClusterPhase = "ReconcilingUpgrade"

	KubeOnePhaseReconcilingMigrate ExternalClusterPhase = "ReconcilingMigrate"

	// ExternalClusterPhaseDeleting status indicates the cluster is being deleted.
	ExternalClusterPhaseDeleting ExternalClusterPhase = "Deleting"

	// ExternalClusterPhaseUnknown Not set.
	ExternalClusterPhaseUnknown ExternalClusterPhase = "Unknown"

	// ExternalClusterPhaseError status indicates the cluster is unusable. Details can be found in the
	// `statusMessage` field.
	ExternalClusterPhaseError ExternalClusterPhase = "Error"

	// ExternalClusterPhaseRuntimeError status indicates cluster runtime error. Details can be found in the
	// `statusMessage` field.
	ExternalClusterPhaseRuntimeError ExternalClusterPhase = "RuntimeError"

	// ExternalClusterPhaseEtcdError status indicates cluster etcd error. Details can be found in the
	// `statusMessage` field.
	ExternalClusterPhaseEtcdError ExternalClusterPhase = "EtcdError"

	// ExternalClusterPhaseKubeClientError status indicates cluster kubeclient. Details can be found in the
	// `statusMessage` field.
	ExternalClusterPhaseKubeClientError ExternalClusterPhase = "KubeClientError"

	// KubeOneExternalClusterPhaseSSHError status indicates cluster ssh error. Details can be found in the
	// `statusMessage` field.
	ExternalClusterPhaseSSHError ExternalClusterPhase = "SSHError"

	// ExternalClusterPhaseConnectionError status indicates cluster connection error. Details can be found in the
	// `statusMessage` field.
	ExternalClusterPhaseConnectionError ExternalClusterPhase = "ConnectionError"

	// ExternalClusterPhaseConfigError status indicates cluster config error. Details can be found in the
	// `statusMessage` field.
	ExternalClusterPhaseConfigError ExternalClusterPhase = "ConfigError"
)

type ExternalClusterBringYourOwnCloudSpec struct{}

type ExternalClusterGKECloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	Name string `json:"name"`
	// ServiceAccount: The Google Cloud Platform Service Account.
	// Can be read from `credentialsReference` instead.
	ServiceAccount string `json:"serviceAccount,omitempty"`
	// Zone: The name of the Google Compute Engine zone
	// (https://cloud.google.com/compute/docs/zones#available) in which the
	// cluster resides.
	Zone string `json:"zone"`
}

type ExternalClusterEKSCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	Name string `json:"name"`
	// The Access key ID used to authenticate against AWS.
	// Can be read from `credentialsReference` instead.
	AccessKeyID string `json:"accessKeyID,omitempty"`
	// The Secret Access Key used to authenticate against AWS.
	// Can be read from `credentialsReference` instead.
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
	Region          string `json:"region"`
	// The Amazon Resource Name (ARN) of the IAM role that provides permissions
	// for the Kubernetes control plane to make calls to Amazon Web Services API
	// operations on your behalf.
	ControlPlaneRoleARN string `json:"roleArn,omitempty"`
	// The VPC associated with your cluster.
	VPCID string `json:"vpcID,omitempty"`
	// The subnets associated with your cluster.
	SubnetIDs []string `json:"subnetIDs,omitempty"`
	// The security groups associated with the cross-account elastic network interfaces
	// that are used to allow communication between your nodes and the Kubernetes
	// control plane.
	SecurityGroupIDs []string `json:"securityGroupIDs,omitempty"`

	// The ARN for an IAM role that should be assumed when handling resources on AWS. It will be used
	// to acquire temporary security credentials using an STS AssumeRole API operation whenever creating an AWS session.
	// required: false
	AssumeRoleARN string `json:"assumeRoleARN,omitempty"` //nolint:tagliatelle
	// An arbitrary string that may be needed when calling the STS AssumeRole API operation.
	// Using an external ID can help to prevent the "confused deputy problem".
	// required: false
	AssumeRoleExternalID string `json:"assumeRoleExternalID,omitempty"`
}

type ExternalClusterAKSCloudSpec struct {
	// CredentialsReference allows referencing a `Secret` resource instead of passing secret data in this spec.
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	Name string `json:"name"`
	// The Azure Active Directory Tenant used for this cluster.
	// Can be read from `credentialsReference` instead.
	TenantID string `json:"tenantID,omitempty"`
	// The Azure Subscription used for this cluster.
	// Can be read from `credentialsReference` instead.
	SubscriptionID string `json:"subscriptionID,omitempty"`
	// The service principal used to access Azure.
	// Can be read from `credentialsReference` instead.
	ClientID string `json:"clientID,omitempty"`
	// The client secret corresponding to the given service principal.
	// Can be read from `credentialsReference` instead.
	ClientSecret string `json:"clientSecret,omitempty"`
	// The geo-location where the resource lives
	Location string `json:"location"`
	// The resource group that will be used to look up and create resources for the cluster in.
	// If set to empty string at cluster creation, a new resource group will be created and this field will be updated to
	// the generated resource group's name.
	ResourceGroup string `json:"resourceGroup"`
}

func (i *ExternalCluster) GetKubeconfigSecretName() string {
	return fmt.Sprintf("%s-%s", ExternalClusterKubeconfigPrefix, i.Name)
}

func (i *ExternalCluster) GetCredentialsSecretName() string {
	// The kubermatic cluster `GetSecretName` method is used to get credential secret name for external cluster
	// The same is used for the external cluster creation when secret is created
	cluster := &Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: i.Name,
		},
		Spec: ClusterSpec{
			Cloud: CloudSpec{},
		},
	}
	cloud := i.Spec.CloudSpec
	if cloud.ProviderName == ExternalClusterBringYourOwnProvider {
		return ""
	}
	if cloud.GKE != nil {
		cluster.Spec.Cloud.GCP = &GCPCloudSpec{}
	}
	if cloud.EKS != nil {
		cluster.Spec.Cloud.AWS = &AWSCloudSpec{}
	}
	if cloud.AKS != nil {
		cluster.Spec.Cloud.Azure = &AzureCloudSpec{}
	}
	return cluster.GetSecretName()
}

func (i *ExternalCluster) GetKubeOneCredentialsSecretName() string {
	providerName := i.Spec.CloudSpec.KubeOne.ProviderName
	return fmt.Sprintf("%s-%s-%s", CredentialPrefix, providerName, i.Name)
}

func (i *ExternalCluster) GetKubeOneSSHSecretName() string {
	return fmt.Sprintf("%s-%s", KubeOneSSHSecretPrefix, i.Name)
}

func (i *ExternalCluster) GetKubeOneManifestSecretName() string {
	return fmt.Sprintf("%s-%s", KubeOneManifestSecretPrefix, i.Name)
}

func (i *ExternalCluster) GetKubeOneNamespaceName() string {
	return fmt.Sprintf("%s-%s", KubeOneNamespacePrefix, i.Name)
}
