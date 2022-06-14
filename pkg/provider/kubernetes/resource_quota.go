package kubernetes

import (
	"context"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceQuotaProvider struct {
	privilegedClient               ctrlruntimeclient.Client
	createMasterImpersonatedClient ImpersonationClient
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

func (p *ResourceQuotaProvider) GetForProject(ctx context.Context, userInfo *provider.UserInfo, projectName string) (*kubermaticv1.ResourceQuota, error) {
	if userInfo == nil {
		return nil, errors.New("a user is missing but required")
	}
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)

	// first: check for quotas with helper labels
	resourceQuotaFilteredList, err := list(ctx, masterImpersonatedClient, map[string]string{
		kubermaticv1.ResourceQuotaSubjectNameLabelKey: projectName,
		kubermaticv1.ResourceQuotaSubjectKindLabelKey: "project",
	})
	if err != nil {
		return nil, err
	}
	if len(resourceQuotaFilteredList.Items) == 1 {
		return &resourceQuotaFilteredList.Items[0], nil
	}

	// second: check kind and name manually in case helper labels are missing
	resourceQuotaFullList, err := list(ctx, masterImpersonatedClient, map[string]string{})
	for _, resourceQuota := range resourceQuotaFullList.Items {
		if resourceQuota.Name == projectName && resourceQuota.Kind == "project" {
			return &resourceQuota, nil
		}
	}
	return nil, utilerrors.NewNotFound("resourcequota for project %s", projectName)
}

func (p *ResourceQuotaProvider) List(ctx context.Context, labelSet map[string]string) (*kubermaticv1.ResourceQuotaList, error) {
	return list(ctx, p.privilegedClient, labelSet)
}

func list(ctx context.Context, client ctrlruntimeclient.Client, labelSet map[string]string) (*kubermaticv1.ResourceQuotaList, error) {
	resourceQuotaList := &kubermaticv1.ResourceQuotaList{}

	selector := labels.SelectorFromSet(labelSet)
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     kubermaticv1.ResourceQuotaNamespace,
		LabelSelector: selector,
	}
	if err := client.List(ctx, resourceQuotaList, listOpts); err != nil {
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
