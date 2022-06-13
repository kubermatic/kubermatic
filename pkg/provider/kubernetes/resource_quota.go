package kubernetes

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceQuotaProvider struct {
	privilegedClient ctrlruntimeclient.Client
}

var _ provider.ResourceQuotaProvider = &ResourceQuotaProvider{}

func NewResourceQuotaProvider(privilegedClient ctrlruntimeclient.Client) *ResourceQuotaProvider {
	return &ResourceQuotaProvider{
		privilegedClient: privilegedClient,
	}
}

func (p *ResourceQuotaProvider) Get(ctx context.Context, name string) (*kubermaticv1.ResourceQuota, error) {
	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := p.privilegedClient.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: kubermaticv1.ResourceQuotaNamespace,
	}, resourceQuota); err != nil {
		return nil, err
	}
	return resourceQuota, nil
}

func (p *ResourceQuotaProvider) List(ctx context.Context, labelSet map[string]string) (*kubermaticv1.ResourceQuotaList, error) {
	resourceQuotaList := &kubermaticv1.ResourceQuotaList{}

	selector := labels.SelectorFromSet(labelSet)
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     kubermaticv1.ResourceQuotaNamespace,
		LabelSelector: selector,
	}
	if err := p.privilegedClient.List(ctx, resourceQuotaList, listOpts); err != nil {
		return nil, err
	}
	return resourceQuotaList, nil
}

func (p *ResourceQuotaProvider) Create(ctx context.Context, subject kubermaticv1.Subject, quota kubermaticv1.ResourceDetails) error {
	rq := &kubermaticv1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels: map[string]string{
				kubermaticv1.ResourceQuotaSubjectNameLabelKey: subject.Name,
				kubermaticv1.ResourceQuotaSubjectKindLabelKey: subject.Kind,
			},
			Namespace: kubermaticv1.ResourceQuotaNamespace,
			Name:      fmt.Sprintf("%s-%s", subject.Kind, subject.Name),
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

func (p *ResourceQuotaProvider) Update(ctx context.Context, name string, newQuota kubermaticv1.ResourceDetails) error {
	rq, err := p.Get(ctx, name)
	if err != nil {
		return err
	}

	rq.Spec.Quota = newQuota

	if err := p.privilegedClient.Update(ctx, rq); err != nil {
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
