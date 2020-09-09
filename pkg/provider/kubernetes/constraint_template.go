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

	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8c.io/kubermatic/v2/pkg/util/restmapper"
)

// ConstraintTemplateProvider struct that holds required components in order manage constraint templates
type ConstraintTemplateProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient impersonationClient
	clientPrivileged               ctrlruntimeclient.Client
	restMapperCache                *restmapper.Cache
}

// NewConstraintTemplateProvider returns a constraint template provider
func NewConstraintTemplateProvider(createMasterImpersonatedClient impersonationClient, client ctrlruntimeclient.Client) (*ConstraintTemplateProvider, error) {
	return &ConstraintTemplateProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               client,
		restMapperCache:                restmapper.New(),
	}, nil
}

// List gets all constraint templates
func (p *ConstraintTemplateProvider) List() (*v1beta1.ConstraintTemplateList, error) {

	constraintTemplates := &v1beta1.ConstraintTemplateList{}
	if err := p.clientPrivileged.List(context.Background(), constraintTemplates); err != nil {
		return nil, fmt.Errorf("failed to list constraint templates: %v", err)
	}

	return constraintTemplates, nil
}

// Get gets a constraint template that
func (p *ConstraintTemplateProvider) Get(name string) (*v1beta1.ConstraintTemplate, error) {

	constraintTemplate := &v1beta1.ConstraintTemplate{}
	if err := p.clientPrivileged.Get(context.Background(), types.NamespacedName{Name: name}, constraintTemplate); err != nil {
		return nil, fmt.Errorf("failed to get constraint template %q : %v", name, err)
	}

	return constraintTemplate, nil
}
