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
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

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

// ClusterFromNamespace filters all Cluster objects and returns the
// one where status.namespaceName matches the given namespace. If no
// such cluster exists, nil is returned (no error).
func ClusterFromNamespace(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (*kubermaticv1.Cluster, error) {
	clusters := kubermaticv1.ClusterList{}
	if err := client.List(ctx, &clusters); err != nil {
		return nil, fmt.Errorf("failed to list Cluster objects: %w", err)
	}

	for i, c := range clusters.Items {
		if c.Status.NamespaceName == namespace {
			return &clusters.Items[i], nil
		}
	}

	return nil, nil
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
