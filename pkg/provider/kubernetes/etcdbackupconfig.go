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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// EtcdBackupConfigProvider struct that holds required components in order manage etcd backup configs
type EtcdBackupConfigProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient impersonationClient
	clientPrivileged             ctrlruntimeclient.Client
}

// NewEtcdBackupConfigProvider returns a constraint provider
func NewEtcdBackupConfigProvider(createSeedImpersonatedClient impersonationClient, client ctrlruntimeclient.Client) *EtcdBackupConfigProvider {
	return &EtcdBackupConfigProvider{
		clientPrivileged:             client,
		createSeedImpersonatedClient: createSeedImpersonatedClient,
	}
}

func EtcdBackupConfigProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter) provider.EtcdBackupConfigProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.EtcdBackupConfigProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		defaultImpersonationClientForSeed := NewImpersonationClient(cfg, mapper)
		clientPrivileged, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
		if err != nil {
			return nil, err
		}
		return NewEtcdBackupConfigProvider(
			defaultImpersonationClientForSeed.CreateImpersonatedClient,
			clientPrivileged,
		), nil
	}
}

func (p *EtcdBackupConfigProvider) Create(userInfo *provider.UserInfo, etcdBackupConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {

	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	err = impersonationClient.Create(context.Background(), etcdBackupConfig)
	return etcdBackupConfig, err
}

func (p *EtcdBackupConfigProvider) CreateUnsecured(etcdBackupConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	err := p.clientPrivileged.Create(context.Background(), etcdBackupConfig)
	return etcdBackupConfig, err
}

func (p *EtcdBackupConfigProvider) Get(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdBackupConfig, error) {

	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	ebc := &kubermaticv1.EtcdBackupConfig{}
	err = impersonationClient.Get(context.Background(), types.NamespacedName{Name: name, Namespace: cluster.Status.NamespaceName}, ebc)
	return ebc, err
}

func (p *EtcdBackupConfigProvider) GetUnsecured(cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdBackupConfig, error) {
	ebc := &kubermaticv1.EtcdBackupConfig{}
	err := p.clientPrivileged.Get(context.Background(), types.NamespacedName{Name: name, Namespace: cluster.Status.NamespaceName}, ebc)
	return ebc, err
}

func (p *EtcdBackupConfigProvider) List(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfigList, error) {

	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	ebcList := &kubermaticv1.EtcdBackupConfigList{}
	err = impersonationClient.List(context.Background(), ebcList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName))
	return ebcList, err
}

func (p *EtcdBackupConfigProvider) ListUnsecured(cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfigList, error) {
	ebcList := &kubermaticv1.EtcdBackupConfigList{}
	err := p.clientPrivileged.List(context.Background(), ebcList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName))
	return ebcList, err
}

func (p *EtcdBackupConfigProvider) Delete(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) error {

	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}

	ebc := &kubermaticv1.EtcdBackupConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	return impersonationClient.Delete(context.Background(), ebc)
}

func (p *EtcdBackupConfigProvider) DeleteUnsecured(cluster *kubermaticv1.Cluster, name string) error {
	ebc := &kubermaticv1.EtcdBackupConfig{
		ObjectMeta: v1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	return p.clientPrivileged.Delete(context.Background(), ebc)
}
