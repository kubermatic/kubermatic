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

package kubernetes

import (
	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

const (
	// NamespacePrefix is the prefix for the cluster namespace
	NamespacePrefix = "cluster-"
)

// kubernetesImpersonationClient gives kubernetes client set that uses user impersonation
type kubernetesImpersonationClient func(impCfg restclient.ImpersonationConfig) (kubernetes.Interface, error)

// kubermaticImpersonationClient gives kubermatic client set that uses user impersonation
type kubermaticImpersonationClient func(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error)

// NamespaceName returns the namespace name for a cluster
func NamespaceName(clusterName string) string {
	return NamespacePrefix + clusterName
}

// createKubermaticImpersonationClientWrapperFromUserInfo is a helper method that spits back kubermatic client that uses user impersonation
func createKubermaticImpersonationClientWrapperFromUserInfo(userInfo *provider.UserInfo, createImpersonationClient kubermaticImpersonationClient) (kubermaticclientv1.KubermaticV1Interface, error) {
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   []string{userInfo.Group},
	}

	return createImpersonationClient(impersonationCfg)
}

// createKubernetesImpersonationClientWrapperFromUserInfo is a helper method that spits back kubernetes client that uses user impersonation
func createKubernetesImpersonationClientWrapperFromUserInfo(userInfo *provider.UserInfo, createImpersonationClient kubernetesImpersonationClient) (kubernetes.Interface, error) {
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   []string{userInfo.Group},
	}

	return createImpersonationClient(impersonationCfg)
}

// NewKubermaticImpersonationClient creates a new default impersonation client
// that knows how to create KubermaticV1Interface client for a impersonated user
//
// Note:
// It is usually not desirable to create many RESTClient thus in the future we might
// consider storing RESTClients in a pool for the given group name
func NewKubermaticImpersonationClient(cfg *restclient.Config) *DefaultKubermaticImpersonationClient {
	return &DefaultKubermaticImpersonationClient{cfg}
}

// DefaultKubermaticImpersonationClient knows how to create impersonated client set
type DefaultKubermaticImpersonationClient struct {
	cfg *restclient.Config
}

// CreateImpersonatedKubermaticClientSet actually creates impersonated kubermatic client set for the given user.
func (d *DefaultKubermaticImpersonationClient) CreateImpersonatedKubermaticClientSet(impCfg restclient.ImpersonationConfig) (kubermaticclientv1.KubermaticV1Interface, error) {
	config := *d.cfg
	config.Impersonate = impCfg
	return kubermaticclientv1.NewForConfig(&config)
}

// NewKubernetesImpersonationClient creates a new default impersonation client
// that knows how to create kubernetes Interface client for a impersonated user
func NewKubernetesImpersonationClient(cfg *restclient.Config) *DefaultKubernetesImpersonationClient {
	return &DefaultKubernetesImpersonationClient{cfg}
}

// DefaultKubermaticImpersonationClient knows how to create impersonated client set
type DefaultKubernetesImpersonationClient struct {
	cfg *restclient.Config
}

// CreateImpersonatedKubernetesClientSet actually creates impersonated kubernetes client set for the given user.
func (d *DefaultKubernetesImpersonationClient) CreateImpersonatedKubernetesClientSet(impCfg restclient.ImpersonationConfig) (kubernetes.Interface, error) {
	config := *d.cfg
	config.Impersonate = impCfg
	return kubernetes.NewForConfig(&config)
}
