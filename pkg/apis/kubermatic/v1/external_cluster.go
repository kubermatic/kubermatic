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

//+genclient
//+genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalCluster is the object representing an external kubernetes cluster.
type ExternalCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ExternalClusterSpec `json:"spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ExternalClusterList specifies a list of external kubernetes clusters
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
}

func (i *ExternalCluster) GetKubeconfigSecretName() string {
	return fmt.Sprintf("kubeconfig-external-cluster-%s", i.Name)
}
