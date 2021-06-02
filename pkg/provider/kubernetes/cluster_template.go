/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"errors"
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterTemplateProvider struct that holds required components in order manage cluster templates
type ClusterTemplateProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient impersonationClient
	clientPrivileged               ctrlruntimeclient.Client
}

// NewClusterTemplateProvider returns a cluster template provider
func NewClusterTemplateProvider(createMasterImpersonatedClient impersonationClient, client ctrlruntimeclient.Client) (*ClusterTemplateProvider, error) {
	return &ClusterTemplateProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               client,
	}, nil
}

func (p *ClusterTemplateProvider) New(userInfo *provider.UserInfo, newClusterTemplate *kubermaticv1.ClusterTemplate, scope, projectID string) (*kubermaticv1.ClusterTemplate, error) {
	if userInfo == nil || newClusterTemplate == nil {
		return nil, errors.New("userInfo and/or cluster is missing but required")
	}
	if scope == "" {
		return nil, errors.New("cluster template scope is missing but required")
	}

	if !userInfo.IsAdmin && scope == kubermaticv1.GlobalClusterTemplateScope {
		return nil, errors.New("the global scope is reserved only for admins")
	}

	if strings.Contains(userInfo.Group, "viewers") && scope != kubermaticv1.UserClusterTemplateScope {
		return nil, fmt.Errorf("viewer is not allowed to create cluster template for the %s scope", scope)
	}

	if scope == kubermaticv1.ProjectClusterTemplateScope && projectID == "" {
		return nil, errors.New("project ID is missing but required")
	}

	if err := p.clientPrivileged.Create(context.Background(), newClusterTemplate); err != nil {
		return nil, err
	}

	return newClusterTemplate, nil

}
