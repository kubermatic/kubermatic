/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// PrivilegedMLAAdminSettingProvider struct that holds required components in order to manage MLAAdminSetting objects.
type PrivilegedMLAAdminSettingProvider struct {
	// privilegedClient is used for admins to interact with MLAAdminSetting objects.
	privilegedClient ctrlruntimeclient.Client
}

func (p *PrivilegedMLAAdminSettingProvider) GetUnsecured(cluster *kubermaticv1.Cluster) (*kubermaticv1.MLAAdminSetting, error) {
	mlaAdminSetting := &kubermaticv1.MLAAdminSetting{}
	if err := p.privilegedClient.Get(context.Background(), types.NamespacedName{
		Name:      resources.DefaultMLAAdminSettingName,
		Namespace: cluster.Status.NamespaceName,
	}, mlaAdminSetting); err != nil {
		return nil, err
	}
	return mlaAdminSetting, nil
}

func (p *PrivilegedMLAAdminSettingProvider) CreateUnsecured(mlaAdminSetting *kubermaticv1.MLAAdminSetting) (*kubermaticv1.MLAAdminSetting, error) {
	err := p.privilegedClient.Create(context.Background(), mlaAdminSetting)
	return mlaAdminSetting, err
}

func (p *PrivilegedMLAAdminSettingProvider) UpdateUnsecured(newMLAAdminSetting *kubermaticv1.MLAAdminSetting) (*kubermaticv1.MLAAdminSetting, error) {
	err := p.privilegedClient.Update(context.Background(), newMLAAdminSetting)
	return newMLAAdminSetting, err
}

func (p *PrivilegedMLAAdminSettingProvider) DeleteUnsecured(cluster *kubermaticv1.Cluster) error {
	return p.privilegedClient.Delete(context.Background(), &kubermaticv1.MLAAdminSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.DefaultMLAAdminSettingName,
			Namespace: cluster.Status.NamespaceName,
		},
	})
}

// NewPrivilegedMLAAdminSettingProvider returns a MLAAdminSetting provider
func NewPrivilegedMLAAdminSettingProvider(privilegedClient ctrlruntimeclient.Client) *PrivilegedMLAAdminSettingProvider {
	return &PrivilegedMLAAdminSettingProvider{
		privilegedClient: privilegedClient,
	}
}

func PrivilegedMLAAdminSettingProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter) provider.PrivilegedMLAAdminSettingProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.PrivilegedMLAAdminSettingProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		privilegedClient, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
		if err != nil {
			return nil, err
		}
		return NewPrivilegedMLAAdminSettingProvider(
			privilegedClient,
		), nil
	}
}
