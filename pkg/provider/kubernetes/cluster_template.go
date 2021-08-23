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
	"net/http"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	kubermaticerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterTemplateProvider struct that holds required components in order manage cluster templates
type ClusterTemplateProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient ImpersonationClient
	clientPrivileged               ctrlruntimeclient.Client
}

// NewClusterTemplateProvider returns a cluster template provider
func NewClusterTemplateProvider(createMasterImpersonatedClient ImpersonationClient, client ctrlruntimeclient.Client) (*ClusterTemplateProvider, error) {
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

func (p *ClusterTemplateProvider) List(userInfo *provider.UserInfo, projectID string) ([]kubermaticv1.ClusterTemplate, error) {
	if userInfo == nil {
		return nil, errors.New("userInfo is missing but required")
	}

	var result []kubermaticv1.ClusterTemplate
	globalUserResult := &kubermaticv1.ClusterTemplateList{}

	if err := p.clientPrivileged.List(context.Background(), globalUserResult); err != nil {
		return nil, err
	}

	for _, template := range globalUserResult.Items {
		switch {
		case template.Labels[kubermaticv1.ClusterTemplateScopeLabelKey] == kubermaticv1.GlobalClusterTemplateScope:
			result = append(result, template)
		case strings.EqualFold(template.Annotations[kubermaticv1.ClusterTemplateUserAnnotationKey], userInfo.Email):
			result = append(result, template)
		case projectID != "" && template.Labels[kubermaticv1.ProjectIDLabelKey] == projectID && template.Labels[kubermaticv1.ClusterTemplateScopeLabelKey] == kubermaticv1.ProjectClusterTemplateScope:
			result = append(result, template)
		}
	}

	return result, nil
}

func (p *ClusterTemplateProvider) Get(userInfo *provider.UserInfo, projectID, templateID string) (*kubermaticv1.ClusterTemplate, error) {
	if userInfo == nil {
		return nil, errors.New("userInfo is missing but required")
	}
	if templateID == "" {
		return nil, errors.New("templateID is missing but required")
	}

	result := &kubermaticv1.ClusterTemplate{}

	if err := p.clientPrivileged.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: templateID}, result); err != nil {
		return nil, err
	}

	if userInfo.IsAdmin {
		return result, nil
	}
	if result.Labels[kubermaticv1.ClusterTemplateScopeLabelKey] == kubermaticv1.GlobalClusterTemplateScope {
		return result, nil
	}

	if result.Labels[kubermaticv1.ClusterTemplateScopeLabelKey] == kubermaticv1.UserClusterTemplateScope && !strings.EqualFold(result.Annotations[kubermaticv1.ClusterTemplateUserAnnotationKey], userInfo.Email) {
		return nil, kubermaticerrors.New(http.StatusForbidden, fmt.Sprintf("user %s can't access template %s", userInfo.Email, templateID))
	}
	if projectID != "" && result.Labels[kubermaticv1.ProjectIDLabelKey] != projectID && result.Labels[kubermaticv1.ClusterTemplateScopeLabelKey] == kubermaticv1.ProjectClusterTemplateScope {
		return nil, kubermaticerrors.New(http.StatusForbidden, fmt.Sprintf("cluster template doesn't belong to the project %s", projectID))
	}

	return result, nil
}

func (p *ClusterTemplateProvider) Delete(userInfo *provider.UserInfo, projectID, templateID string) error {
	if userInfo == nil {
		return errors.New("userInfo is missing but required")
	}
	if templateID == "" {
		return errors.New("templateID is missing but required")
	}

	result, err := p.Get(userInfo, projectID, templateID)
	if err != nil {
		return err
	}

	// only admin can delete global templates
	if !userInfo.IsAdmin && result.Labels[kubermaticv1.ClusterTemplateScopeLabelKey] == kubermaticv1.GlobalClusterTemplateScope {
		return kubermaticerrors.New(http.StatusForbidden, fmt.Sprintf("user %s can't delete template %s", userInfo.Email, templateID))
	}

	return p.clientPrivileged.Delete(context.Background(), result)
}
