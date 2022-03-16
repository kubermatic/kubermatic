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

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:JSONPath=".spec.humanReadableName",name="HumanReadableName",type="string"
// +kubebuilder:printcolumn:JSONPath=".metadata.creationTimestamp",name="Age",type="date"

// ExternalCluster is the object representing an external kubernetes cluster.
type ExternalCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ExternalClusterSpec `json:"spec,omitempty"`
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

	KubeconfigReference *providerconfig.GlobalSecretKeySelector `json:"kubeconfigReference,omitempty"`
	CloudSpec           *ExternalClusterCloudSpec               `json:"cloudSpec,omitempty"`
}

// ExternalClusterCloudSpec mutually stores access data to a cloud provider.
type ExternalClusterCloudSpec struct {
	GKE     *ExternalClusterGKECloudSpec     `json:"gke,omitempty"`
	EKS     *ExternalClusterEKSCloudSpec     `json:"eks,omitempty"`
	AKS     *ExternalClusterAKSCloudSpec     `json:"aks,omitempty"`
	KubeOne *ExternalClusterKubeOneCloudSpec `json:"kubeone,omitempty"`
}

type KubeOneExternalClusterState string

const (
	// PROVISIONING state indicates the cluster is being imported.
	PROVISIONING KubeOneExternalClusterState = "PROVISIONING"

	// RUNNING state indicates the cluster is fully usable.
	RUNNING KubeOneExternalClusterState = "RUNNING"

	// RECONCILING state indicates that some work is actively being done on the cluster, such as upgrading the master or
	// node software. Details can be found in the `StatusMessage` field.
	RECONCILING KubeOneExternalClusterState = "RECONCILING"

	// DELETING state indicates the cluster is being deleted.
	DELETING KubeOneExternalClusterState = "DELETING"

	// UNKNOWN Not set.
	UNKNOWN KubeOneExternalClusterState = "UNKNOWN"

	// ERROR state indicates the cluster is unusable. It will be automatically deleted. Details can be found in the
	// `statusMessage` field.
	ERROR KubeOneExternalClusterState = "ERROR"
)

// ExternalClusterStatus defines the kubeone external cluster status.
type KubeOneExternalClusterStatus struct {
	State         KubeOneExternalClusterState `json:"state"`
	StatusMessage string                      `json:"statusMessage,omitempty"`
}

type ExternalClusterKubeOneCloudSpec struct {
	Status KubeOneExternalClusterStatus `json:"status,omitempty"`
	// ProviderName is the name of the cloud provider used, one of
	// "aws", "azure", "digitalocean", "gcp",
	// "hetzner", "nutanix", "openstack", "packet", "vsphere" KubeOne natively-supported providers
	ProviderName         string                                 `json:"providerName"`
	CredentialsReference providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`
	SSHReference         providerconfig.GlobalSecretKeySelector `json:"sshReference"`
	ManifestReference    providerconfig.GlobalSecretKeySelector `json:"manifestReference"`
}

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
