package kubernetes

import (
	"context"
	"crypto/sha256"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func (p *ResourceQuotaProvider) List(ctx context.Context) (*kubermaticv1.ResourceQuotaList, error) {
	resourceQuotaList := &kubermaticv1.ResourceQuotaList{}
	if err := p.privilegedClient.List(ctx, resourceQuotaList, ctrlruntimeclient.InNamespace(ResourceQuotaNamespace)); err != nil {
		return nil, err
	}
	return resourceQuotaList, nil
}

func (p *ResourceQuotaProvider) Create(ctx context.Context, subject kubermaticv1.Subject, quota kubermaticv1.ResourceDetails) error {
	name := sha256.Sum256([]byte(fmt.Sprintf("%s-%s", subject.Kind, subject.Name)))
	rq := &kubermaticv1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      map[string]string{},
			Namespace:   ResourceQuotaNamespace,
			Name:        fmt.Sprintf("%x", name),
		},
		Spec: kubermaticv1.ResourceQuotaSpec{
			Subject: subject,
			Quota:   quota,
		},
	}

	if err := p.privilegedClient.Create(ctx, rq); err != nil {
		return err
	}
	return nil
}

func (p *ResourceQuotaProvider) Update(ctx context.Context, resourceQuota *kubermaticv1.ResourceQuota) error {
	if err := p.privilegedClient.Update(ctx, resourceQuota); err != nil {
		return err
	}
	return nil
}

func (p *ResourceQuotaProvider) Delete(ctx context.Context, name string) error {
	rq, err := p.Get(ctx, name)
	if err != nil {
		return err
	}
	if err := p.privilegedClient.Delete(ctx, rq); err != nil {
		return err
	}
	return nil
}
