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

	// ExternalClusterResourceName represents "Resource" defined in Kubernetes
	ExternalClusterResourceName = "externalclusters"

	// ExternalClusterKind represents "Kind" defined in Kubernetes
	ExternalClusterKind = "ExternalCluster"
)

// +kubebuilder:resource:scope=Cluster
// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ExternalCluster is the object representing an external kubernetes cluster.
type ExternalCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ExternalClusterSpec `json:"spec,omitempty"`
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
	GKE *ExternalClusterGKECloudSpec `json:"gke,omitempty"`
	EKS *ExternalClusterEKSCloudSpec `json:"eks,omitempty"`
}

type ExternalClusterGKECloudSpec struct {
	Name                 string                                  `json:"name"`
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`
	Zone                 string                                  `json:"zone,omitempty"`
}

type ExternalClusterEKSCloudSpec struct {
	Name                 string                                  `json:"name"`
	CredentialsReference *providerconfig.GlobalSecretKeySelector `json:"credentialsReference,omitempty"`
	Region               string                                  `json:"region"`
}

func (i *ExternalCluster) GetKubeconfigSecretName() string {
	return fmt.Sprintf("kubeconfig-external-cluster-%s", i.Name)
}

// +kubebuilder:object:generate=true
// +kubebuilder:object:root=true

// ExternalClusterList specifies a list of external kubernetes clusters
type ExternalClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ExternalCluster `json:"items"`
}
