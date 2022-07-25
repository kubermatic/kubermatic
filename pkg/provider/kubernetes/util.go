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
	"strings"

	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/api/meta"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// NamespacePrefix is the prefix for the cluster namespace.
	NamespacePrefix = "cluster-"
)

// ImpersonationClient gives runtime controller client that uses user impersonation.
type ImpersonationClient func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error)

// NamespaceName returns the namespace name for a cluster.
func NamespaceName(clusterName string) string {
	return NamespacePrefix + clusterName
}

// ClusterNameFromNamespace returns name of a cluster from the cluster namespace.
func ClusterNameFromNamespace(namespace string) string {
	if !strings.HasPrefix(namespace, NamespacePrefix) {
		return ""
	}

	return strings.TrimPrefix(namespace, NamespacePrefix)
}

// createImpersonationClientWrapperFromUserInfo is a helper method that spits back controller runtime client that uses user impersonation.
func createImpersonationClientWrapperFromUserInfo(userInfo *provider.UserInfo, createImpersonationClient ImpersonationClient) (ctrlruntimeclient.Client, error) {
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   userInfo.Groups,
	}

	return createImpersonationClient(impersonationCfg)
}

// NewImpersonationClient creates a new default impersonation client
// that knows how to create Interface client for a impersonated user.
func NewImpersonationClient(cfg *restclient.Config, restMapper meta.RESTMapper) *DefaultImpersonationClient {
	return &DefaultImpersonationClient{
		cfg:        cfg,
		restMapper: restMapper,
	}
}

// DefaultImpersonationClient knows how to create impersonated client set.
type DefaultImpersonationClient struct {
	cfg        *restclient.Config
	restMapper meta.RESTMapper
}

// CreateImpersonatedClient actually creates impersonated client set for the given user.
func (d *DefaultImpersonationClient) CreateImpersonatedClient(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
	config := *d.cfg
	config.Impersonate = impCfg

	return ctrlruntimeclient.New(&config, ctrlruntimeclient.Options{Mapper: d.restMapper})
}
