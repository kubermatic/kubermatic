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
	KubeOneSpec         *ExternalClusterKubeOneSpec             `json:"kubeoneSpec,omitempty"`
}

type ExternalClusterKubeOneSpec struct {
	SSHReference      *providerconfig.GlobalSecretKeySelector `json:"sshReference,omitempty"`
	ManifestReference *providerconfig.GlobalSecretKeySelector `json:"manifestReference,omitempty"`
	CloudSpec         *KubeOneCloudSpec                       `json:"cloudSpec,omitempty"`
}

type KubeOneCloudSpec struct {
	AWS          *KubeOneAWSCloudSpec          `json:"aws,omitempty"`
	GCP          *KubeOneGCPCloudSpec          `json:"gcp,omitempty"`
	Azure        *KubeOneAzureCloudSpec        `json:"azure,omitempty"`
	Digitalocean *KubeOneDigitaloceanCloudSpec `json:"digitalocean,omitempty"`
	Openstack    *KubeOneOpenstackCloudSpec    `json:"openstack,omitempty"`
	Packet       *KubeOnePacketCloudSpec       `json:"packet,omitempty"`
	Hetzner      *KubeOneHetznerCloudSpec      `json:"hetzner,omitempty"`
	VSphere      *KubeOneVSphereCloudSpec      `json:"vsphere,omitempty"`
	Nutanix      *KubeOneNutanixCloudSpec      `json:"nutanix,omitempty"`
}

// KubeOneAWSCloudSpec specifies access data to Amazon Web Services.
type KubeOneAWSCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	AccessKeyID     string `json:"accessKeyID,omitempty"`
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
}

// KubeOneGCPCloudSpec specifies access data to GCP.
type KubeOneGCPCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	ServiceAccount string `json:"serviceAccount,omitempty"`
}

// KubeOneAzureCloudSpec specifies access credentials to Azure cloud.
type KubeOneAzureCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	TenantID       string `json:"tenantID,omitempty"`
	SubscriptionID string `json:"subscriptionID,omitempty"`
	ClientID       string `json:"clientID,omitempty"`
	ClientSecret   string `json:"clientSecret,omitempty"`
}

// KubeOneDigitaloceanCloudSpec specifies access data to DigitalOcean.
type KubeOneDigitaloceanCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	Token string `json:"token,omitempty"` // Token is used to authenticate with the DigitalOcean API.
}

// KubeOneOpenstackCloudSpec specifies access data to an OpenStack cloud.
type KubeOneOpenstackCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	AuthURL  string `json:"authURL,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`

	// project, formally known as tenant.
	Project string `json:"project"`
	// project id, formally known as tenantID.
	ProjectID string `json:"projectID"`

	Domain string `json:"domain"`
}

// KubeOneVSphereCloudSpec credentials represents a credential for accessing vSphere.
type KubeOneVSphereCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	Server   string `json:"server"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// KubeOnePacketCloudSpec specifies access data to a Packet cloud.
type KubeOnePacketCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	APIKey    string `json:"apiKey"`
	ProjectID string `json:"projectID"`
}

// KubeOneHetznerCloudSpec specifies access data to hetzner cloud.
type KubeOneHetznerCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	// Token is used to authenticate with the Hetzner cloud API.
	Token string `json:"token,omitempty"`
}

// KubeOneNutanixCloudSpec specifies the access data to Nutanix.
type KubeOneNutanixCloudSpec struct {
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference"`

	// ClusterName is the Nutanix cluster that this user cluster will be deployed to.
	ClusterName   string `json:"clusterName,omitempty"`
	AllowInsecure bool   `json:"allowInsecure,omitempty"`
	ProxyURL      string `json:"proxyURL,omitempty"`

	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	// The Nutanix API (Prism Central) endpoint
	Endpoint string `json:"endpoint"`
	// The Nutanix API (Prism Central) port
	Port string `json:"port"`

	// Prism Element Username for csi driver
	PrismElementUsername string `json:"elementUsername,omitempty"`

	// Prism Element Password for csi driver
	PrismElementPassword string `json:"elementPassword,omitempty"`

	// Prism Element Endpoint to access Nutanix Prism Element for csi driver
	PrismElementEndpoint string `json:"elementEndpoint"`
}

// ExternalClusterCloudSpec mutually stores access data to a cloud provider.
type ExternalClusterCloudSpec struct {
	GKE *ExternalClusterGKECloudSpec `json:"gke,omitempty"`
	EKS *ExternalClusterEKSCloudSpec `json:"eks,omitempty"`
	AKS *ExternalClusterAKSCloudSpec `json:"aks,omitempty"`
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
	return cluster.GetSecretName()
}
