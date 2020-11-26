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

package resources

import (
	"context"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/certificates/triple"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Openshift data contains all data required for Openshift control plane components
// It should be as small as possible
type openshiftData interface {
	Cluster() *kubermaticv1.Cluster
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetPodTemplateLabelsWithContext(context.Context, string, []corev1.Volume, map[string]string) (map[string]string, error)
	GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error)
	GetApiserverExternalNodePort(context.Context) (int32, error)
	NodePortRange(context.Context) string
	ClusterIPByServiceName(name string) (string, error)
	ImageRegistry(string) string
	NodeAccessNetwork() string
	GetClusterRef() metav1.OwnerReference
	GetRootCA() (*triple.KeyPair, error)
	GetRootCAWithContext(context.Context) (*triple.KeyPair, error)
	DC() *kubermaticv1.Datacenter
	EtcdDiskSize() resource.Quantity
	NodeLocalDNSCacheEnabled() bool
	KubermaticAPIImage() string
	KubermaticDockerTag() string
	DNATControllerImage() string
	DNATControllerTag() string
	GetOauthExternalNodePort() (int32, error)
	ExternalURL() string
	Seed() *kubermaticv1.Seed
}
