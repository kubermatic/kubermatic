//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package resourcequotas

import (
	"context"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"

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
		Namespace: resources.KubermaticNamespace,
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
		Namespace: resources.KubermaticNamespace,
	}, resourceQuota); err != nil {
		return nil, err
	}
	return resourceQuota, nil
}

func (p *ResourceQuotaProvider) ListUnsecured(ctx context.Context, labelSet map[string]string) (*kubermaticv1.ResourceQuotaList, error) {
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace:     resources.KubermaticNamespace,
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
			Namespace: resources.KubermaticNamespace,
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
