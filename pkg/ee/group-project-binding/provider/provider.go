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

package provider

import (
	"context"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type GroupProjectBindingProvider struct {
	createMasterImpersonatedClient kubernetes.ImpersonationClient
	clientPrivileged               ctrlruntimeclient.Client
}

var _ provider.GroupProjectBindingProvider = &GroupProjectBindingProvider{}

func NewGroupProjectBindingProvider(createMasterImpersonatedClient kubernetes.ImpersonationClient, clientPrivileged ctrlruntimeclient.Client) *GroupProjectBindingProvider {
	return &GroupProjectBindingProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               clientPrivileged,
	}
}

func (p *GroupProjectBindingProvider) List(ctx context.Context, userInfo *provider.UserInfo, projectID string) ([]kubermaticv1.GroupProjectBinding, error) {
	if userInfo == nil {
		return nil, errors.New("a user is missing but required")
	}

	projectBindings := &kubermaticv1.GroupProjectBindingList{}
	projectReq, err := labels.NewRequirement(kubermaticv1.ProjectIDLabelKey, selection.Equals, []string{projectID})
	if err != nil {
		return nil, fmt.Errorf("failed to construct project label selector: %w", err)
	}
	listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: labels.NewSelector().Add(*projectReq)}
	if err := p.clientPrivileged.List(ctx, projectBindings, listOpts); err != nil {
		return nil, err
	}

	if len(projectBindings.Items) > 0 {
		// TODO: once we merge group support, instead of kube api request, read permissions from userInfo.
		// Fetch first binding with kube API to ensure user has permissions
		_, err := p.Get(ctx, userInfo, projectBindings.Items[0].Name)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, nil
	}

	return projectBindings.Items, nil
}

func (p *GroupProjectBindingProvider) Get(ctx context.Context, userInfo *provider.UserInfo, name string) (*kubermaticv1.GroupProjectBinding, error) {
	masterImpersonatedClient, err := p.getImpersonatedClient(userInfo)
	if err != nil {
		return nil, err
	}

	binding := &kubermaticv1.GroupProjectBinding{}
	if err := masterImpersonatedClient.Get(ctx, types.NamespacedName{Name: name}, binding); err != nil {
		return nil, err
	}

	return binding, nil
}

func (p *GroupProjectBindingProvider) Create(ctx context.Context, userInfo *provider.UserInfo, binding *kubermaticv1.GroupProjectBinding) error {
	masterImpersonatedClient, err := p.getImpersonatedClient(userInfo)
	if err != nil {
		return err
	}

	return masterImpersonatedClient.Create(ctx, binding)
}

func (p *GroupProjectBindingProvider) Patch(ctx context.Context, userInfo *provider.UserInfo, oldBinding, newBinding *kubermaticv1.GroupProjectBinding) error {
	masterImpersonatedClient, err := p.getImpersonatedClient(userInfo)
	if err != nil {
		return err
	}

	return masterImpersonatedClient.Patch(ctx, newBinding, ctrlruntimeclient.MergeFrom(oldBinding))
}

func (p *GroupProjectBindingProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, name string) error {
	binding, err := p.Get(ctx, userInfo, name)
	if err != nil {
		return err
	}

	masterImpersonatedClient, err := p.getImpersonatedClient(userInfo)
	if err != nil {
		return err
	}

	return masterImpersonatedClient.Delete(ctx, binding)
}

func (p *GroupProjectBindingProvider) getImpersonatedClient(userInfo *provider.UserInfo) (ctrlruntimeclient.Client, error) {
	if userInfo == nil {
		return nil, errors.New("a user is missing but required")
	}
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: userInfo.Email,
		Groups:   userInfo.Groups,
	}
	return p.createMasterImpersonatedClient(impersonationCfg)
}
