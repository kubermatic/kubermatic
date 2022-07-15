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

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ExternalClusterResourceName represents "Resource" defined in Kubernetes.
	ExternalClusterResourceName = "externalclusters"

	// ExternalClusterKind represents "Kind" defined in Kubernetes.
	ExternalClusterKind = "ExternalCluster"
)

// +kubebuilder:validation:Enum=gke;aks;eks;kubeone

type ExternalClusterProvider string

const (
	ExternalClusterGKEProvider     ExternalClusterProvider = "gke"
	ExternalClusterEKSProvider     ExternalClusterProvider = "eks"
	ExternalClusterAKSProvider     ExternalClusterProvider = "aks"
	ExternalClusterKubeOneProvider ExternalClusterProvider = "kubeone"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".spec.humanReadableName",name="HumanReadableName",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.cloud.providerName",name="Provider",type="string"
// +kubebuilder:printcolumn:JSONPath=".spec.pause",name="Paused",type="boolean"
// +kubebuilder:printcolumn:JSONPath=".status.condition.phase",name="Phase",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// ExternalCluster is the object representing an external kubernetes cluster.
type ExternalCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ExternalClusterSpec   `json:"spec"`
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
	// ProviderName is the name of the cloud provider used, one of
	// "aws", "azure", "digitalocean", "gcp",
	// "hetzner", "nutanix", "openstack", "packet", "vsphere" KubeOne natively-supported providers
	ProviderName         string                                 `json:"providerName"`
	CredentialsReference providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`
	SSHReference         providerconfig.GlobalSecretKeySelector `json:"sshReference"`
	ManifestReference    providerconfig.GlobalSecretKeySelector `json:"manifestReference"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ExternalClusterList specifies a list of external kubernetes clusters.
type ExternalClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ExternalCluster `json:"items"`
}

// ExternalClusterSpec specifies the data for a new external kubernetes cluster.
type ExternalClusterSpec struct {
	// HumanReadableName is the cluster name provided by the user
	HumanReadableName string `json:"humanReadableName"`

	// KubeconfigReference is reference to cluster Kubeconfig
	KubeconfigReference *providerconfig.GlobalSecretKeySelector `json:"kubeconfigReference,omitempty"`
	// CloudSpec contains provider specific fields
	CloudSpec *ExternalClusterCloudSpec `json:"cloudSpec,omitempty"`
	// If this is set to true, the cluster will not be reconciled by KKP.
	// This indicates that the user needs to do some action to resolve the pause.
	Pause bool `json:"pause"`
	// PauseReason is the reason why the cluster is not being managed. This field is for informational
	// purpose only and can be set by a user or a controller to communicate the reason for pausing the cluster.
	PauseReason string `json:"pauseReason,omitempty"`
}

// ExternalClusterCloudSpec mutually stores access data to a cloud provider.
type ExternalClusterCloudSpec struct {
	ProviderName ExternalClusterProvider          `json:"providerName"`
	GKE          *ExternalClusterGKECloudSpec     `json:"gke,omitempty"`
	EKS          *ExternalClusterEKSCloudSpec     `json:"eks,omitempty"`
	AKS          *ExternalClusterAKSCloudSpec     `json:"aks,omitempty"`
	KubeOne      *ExternalClusterKubeOneCloudSpec `json:"kubeone,omitempty"`
}

type ExternalClusterPhase string

const (
	// ExternalClusterPhaseProvisioning status indicates the cluster is being imported.
	ExternalClusterPhaseProvisioning ExternalClusterPhase = "Provisioning"

	// ExternalClusterPhaseRunning status indicates the cluster is fully usable.
	ExternalClusterPhaseRunning ExternalClusterPhase = "Running"

	// ExternalClusterPhaseReconciling status indicates that some work is actively being done on the cluster, such as upgrading the master or
	// node software. Details can be found in the `StatusMessage` field.
	ExternalClusterPhaseReconciling ExternalClusterPhase = "Reconciling"

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

type ExternalClusterGKECloudSpec struct {
	Name                 string                                  `json:"name"`
	ServiceAccount       string                                  `json:"serviceAccount"`
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`
	Zone                 string                                  `json:"zone,omitempty"`
}

type ExternalClusterEKSCloudSpec struct {
	Name                 string                                  `json:"name"`
	AccessKeyID          string                                  `json:"accessKeyID"`
	SecretAccessKey      string                                  `json:"secretAccessKey"`
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`
	Region               string                                  `json:"region"`
}

type ExternalClusterAKSCloudSpec struct {
	Name                 string                                  `json:"name"`
	TenantID             string                                  `json:"tenantID"`
	SubscriptionID       string                                  `json:"subscriptionID"`
	ClientID             string                                  `json:"clientID"`
	ClientSecret         string                                  `json:"clientSecret"`
	ResourceGroup        string                                  `json:"resourceGroup"`
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`
}

func (i *ExternalCluster) GetKubeconfigSecretName() string {
	return fmt.Sprintf("kubeconfig-external-cluster-%s", i.Name)
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
	if cloud == nil {
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
