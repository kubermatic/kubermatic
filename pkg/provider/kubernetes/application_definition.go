package kubernetes

import (
	"context"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// After refactoring the master-rbac controller, this provider can make use of the impersonated
// master client (see 3ed4d3c3036cf6c4941d8703c2067fa655a20a74 for implementation).
// However at the writing of this, it would require a large overhaul in the master-rbac-controller
// for it to handle kubernetes cluster-scoped objects, that have no reference to kubermatic cluster
// and projects. Therefore it was decided for now to make use of the master client directly for now.
type ApplicationDefinitionProvider struct {
	priviledgedClient ctrlruntimeclient.Client
}

var _ provider.ApplicationDefinitionProvider = &ApplicationDefinitionProvider{}

func NewApplicationDefinitionProvider(priviledgedClient ctrlruntimeclient.Client) *ApplicationDefinitionProvider {
	return &ApplicationDefinitionProvider{
		priviledgedClient: priviledgedClient,
	}
}

func (p *ApplicationDefinitionProvider) ListUnsecured(ctx context.Context) (*appskubermaticv1.ApplicationDefinitionList, error) {
	appDefList := &appskubermaticv1.ApplicationDefinitionList{}
	if err := p.priviledgedClient.List(ctx, appDefList); err != nil {
		return nil, err
	}
	return appDefList, nil
}
