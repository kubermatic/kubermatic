package resourcequotas

import (
	"context"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceQuotaProvider struct {
	privilegedClient               ctrlruntimeclient.Client
	createMasterImpersonatedClient kubernetes.ImpersonationClient
}

var _ provider.ResourceQuotaProvider = &ResourceQuotaProvider{}

func NewResourceQuotaProvider(createMasterImpersonatedClient kubernetes.ImpersonationClient, privilegedClient ctrlruntimeclient.Client) *ResourceQuotaProvider {
	return &ResourceQuotaProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		privilegedClient:               privilegedClient,
	}
}

func (p *ResourceQuotaProvider) GetUnsecured(ctx context.Context, name string) (*kubermaticv1.ResourceQuota, error) {
	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := p.privilegedClient.Get(ctx, types.NamespacedName{
		Name:      name,
		Namespace: kubermaticv1.ResourceQuotaNamespace,
	}, resourceQuota); err != nil {
		return nil, err
	}
	return resourceQuota, nil
}

func (p *ResourceQuotaProvider) Get(ctx context.Context, userInfo *provider.UserInfo, name, kind string) (*kubermaticv1.ResourceQuota, error) {
	if userInfo == nil {
		return nil, errors.New("a user is missing but required")
	}

	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   []string{userInfo.Group},
	}
	masterImpersonatedClient, err := p.createMasterImpersonatedClient(impersonationCfg)
	if err != nil {
		return nil, err
	}

	resourceQuota := &kubermaticv1.ResourceQuota{}
	if err := masterImpersonatedClient.Get(ctx, types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", kind, name),
		Namespace: kubermaticv1.ResourceQuotaNamespace,
	}, resourceQuota); err != nil {
		return nil, err
	}
	return resourceQuota, nil
}

func (p *ResourceQuotaProvider) ListUnsecured(ctx context.Context, labelSet map[string]string) (*kubermaticv1.ResourceQuotaList, error) {
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     kubermaticv1.ResourceQuotaNamespace,
		LabelSelector: labels.SelectorFromSet(labelSet),
	}
	resourceQuotaList := &kubermaticv1.ResourceQuotaList{}
	if err := p.privilegedClient.List(ctx, resourceQuotaList, listOpts); err != nil {
		return nil, err
	}
	return resourceQuotaList, nil
}

func (p *ResourceQuotaProvider) CreateUnsecured(ctx context.Context, subject kubermaticv1.Subject, quota kubermaticv1.ResourceDetails) error {
	rq := &kubermaticv1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			// Add labels for easier filtering
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

func (p *ResourceQuotaProvider) UpdateUnsecured(ctx context.Context, name string, newQuota kubermaticv1.ResourceDetails) error {
	rq, err := p.GetUnsecured(ctx, name)
	if err != nil {
		return err
	}

	rq.Spec.Quota = newQuota

	if err := p.privilegedClient.Update(ctx, rq); err != nil {
		return err
	}

	return nil
}

func (p *ResourceQuotaProvider) DeleteUnsecured(ctx context.Context, name string) error {
	rq, err := p.GetUnsecured(ctx, name)
	if err != nil {
		return err
	}

	if err := p.privilegedClient.Delete(ctx, rq); err != nil {
		return err
	}

	return nil
}
