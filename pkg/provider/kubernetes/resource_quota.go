package kubernetes

import (
	"context"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceQuotaProvider struct {
	privilegedClient ctrlruntimeclient.Client
}

var _ provider.ResourceQuotaProvider = &ResourceQuotaProvider{}

const ResourceQuotaNamespace = "kubermatic"

func NewResourceQuotaProvider(privilegedClient ctrlruntimeclient.Client) *ResourceQuotaProvider {
	return &ResourceQuotaProvider{
		privilegedClient: privilegedClient,
	}
}

func (p *ResourceQuotaProvider) Get(ctx context.Context, name string) (*kubermaticv1.ResourceQuota, error) {
	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := p.privilegedClient.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: ResourceQuotaNamespace,
	}, resourceQuota); err != nil {
		return nil, err
	}
	return resourceQuota, nil
}
