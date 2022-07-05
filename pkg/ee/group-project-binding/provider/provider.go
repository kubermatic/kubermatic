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

	v1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type GroupProjectBindingProvider struct {
	createMasterImpersonatedClient kubernetes.ImpersonationClient
}

var _ provider.GroupProjectBindingProvider = &GroupProjectBindingProvider{}

func NewGroupProjectBindingProvider(createMasterImpersonatedClient kubernetes.ImpersonationClient) *GroupProjectBindingProvider {
	return &GroupProjectBindingProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
	}
}

func (p *GroupProjectBindingProvider) List(ctx context.Context, userInfo *provider.UserInfo) ([]v1.GroupProjectBinding, error) {
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

	bindingList := &v1.GroupProjectBindingList{}
	listOpts := &ctrlruntimeclient.ListOptions{}

	if err := masterImpersonatedClient.List(ctx, bindingList, listOpts); err != nil {
		return nil, err
	}
	return bindingList.Items, nil
}
