/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// After refactoring the master-rbac controller, this provider can make use of
// the impersonated master client (see
// https://github.com/kubermatic/kubermatic/pull/10341/commits/42a2df1820e06a4eec354ced7f07e4d3833e5b70
// for implementation). However at the writing of this, it would require a large
// overhaul in the master-rbac-controller for it to handle kubernetes
// cluster-scoped objects, that have no reference to kubermatic clusters and/or
// projects. Therefore it was decided to make use of the master client directly
// for now.
type ApplicationDefinitionProvider struct {
	privilegedClient ctrlruntimeclient.Client
}

var _ provider.ApplicationDefinitionProvider = &ApplicationDefinitionProvider{}

func NewApplicationDefinitionProvider(privilegedClient ctrlruntimeclient.Client) *ApplicationDefinitionProvider {
	return &ApplicationDefinitionProvider{
		privilegedClient: privilegedClient,
	}
}

func (p *ApplicationDefinitionProvider) ListUnsecured(ctx context.Context) (*appskubermaticv1.ApplicationDefinitionList, error) {
	appDefList := &appskubermaticv1.ApplicationDefinitionList{}
	if err := p.privilegedClient.List(ctx, appDefList); err != nil {
		return nil, err
	}
	return appDefList, nil
}

func (p *ApplicationDefinitionProvider) GetUnsecured(ctx context.Context, appDefName string) (*appskubermaticv1.ApplicationDefinition, error) {
	appDef := &appskubermaticv1.ApplicationDefinition{}
	if err := p.privilegedClient.Get(ctx, types.NamespacedName{Name: appDefName}, appDef); err != nil {
		return nil, err
	}
	return appDef, nil
}
