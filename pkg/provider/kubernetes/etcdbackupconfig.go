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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// EtcdBackupConfigProvider struct that holds required components in order manage etcd backup configs.
type EtcdBackupConfigProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient ImpersonationClient
	clientPrivileged             ctrlruntimeclient.Client
}

var _ provider.EtcdBackupConfigProvider = &EtcdBackupConfigProvider{}
var _ provider.PrivilegedEtcdBackupConfigProvider = &EtcdBackupConfigProvider{}

// NewEtcdBackupConfigProvider returns a constraint provider.
func NewEtcdBackupConfigProvider(createSeedImpersonatedClient ImpersonationClient, client ctrlruntimeclient.Client) *EtcdBackupConfigProvider {
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

func (p *EtcdBackupConfigProvider) Create(ctx context.Context, userInfo *provider.UserInfo, etcdBackupConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	err = impersonationClient.Create(ctx, etcdBackupConfig)
	return etcdBackupConfig, err
}

func (p *EtcdBackupConfigProvider) CreateUnsecured(ctx context.Context, etcdBackupConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	err := p.clientPrivileged.Create(ctx, etcdBackupConfig)
	return etcdBackupConfig, err
}

func (p *EtcdBackupConfigProvider) Get(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdBackupConfig, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	ebc := &kubermaticv1.EtcdBackupConfig{}
	err = impersonationClient.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Status.NamespaceName}, ebc)
	return ebc, err
}

func (p *EtcdBackupConfigProvider) GetUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdBackupConfig, error) {
	ebc := &kubermaticv1.EtcdBackupConfig{}
	err := p.clientPrivileged.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Status.NamespaceName}, ebc)
	return ebc, err
}

func (p *EtcdBackupConfigProvider) List(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfigList, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	ebcList := &kubermaticv1.EtcdBackupConfigList{}
	err = impersonationClient.List(ctx, ebcList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName))
	return ebcList, err
}

func (p *EtcdBackupConfigProvider) ListUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfigList, error) {
	ebcList := &kubermaticv1.EtcdBackupConfigList{}
	err := p.clientPrivileged.List(ctx, ebcList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName))
	return ebcList, err
}

func (p *EtcdBackupConfigProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) error {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}

	ebc := &kubermaticv1.EtcdBackupConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	return impersonationClient.Delete(ctx, ebc)
}

func (p *EtcdBackupConfigProvider) DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, name string) error {
	ebc := &kubermaticv1.EtcdBackupConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	return p.clientPrivileged.Delete(ctx, ebc)
}

func (p *EtcdBackupConfigProvider) Patch(ctx context.Context, userInfo *provider.UserInfo, oldConfig, newConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	err = impersonationClient.Patch(ctx, newConfig, ctrlruntimeclient.MergeFrom(oldConfig))
	return newConfig, err
}

func (p *EtcdBackupConfigProvider) PatchUnsecured(ctx context.Context, oldConfig, newConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	err := p.clientPrivileged.Patch(ctx, newConfig, ctrlruntimeclient.MergeFrom(oldConfig))
	return newConfig, err
}

// EtcdBackupConfigProjectProvider struct that holds required components in order manage etcd backup backupConfigs across projects.
type EtcdBackupConfigProjectProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClients map[string]ImpersonationClient
	clientsPrivileged             map[string]ctrlruntimeclient.Client
}

var _ provider.EtcdBackupConfigProjectProvider = &EtcdBackupConfigProjectProvider{}
var _ provider.PrivilegedEtcdBackupConfigProjectProvider = &EtcdBackupConfigProjectProvider{}

// NewEtcdBackupConfigProjectProvider returns an etcd backupConfig global provider.
func NewEtcdBackupConfigProjectProvider(createSeedImpersonatedClients map[string]ImpersonationClient, clients map[string]ctrlruntimeclient.Client) *EtcdBackupConfigProjectProvider {
	return &EtcdBackupConfigProjectProvider{
		clientsPrivileged:             clients,
		createSeedImpersonatedClients: createSeedImpersonatedClients,
	}
}

func EtcdBackupConfigProjectProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter) provider.EtcdBackupConfigProjectProviderGetter {
	return func(seeds map[string]*kubermaticv1.Seed) (provider.EtcdBackupConfigProjectProvider, error) {
		clientsPrivileged := make(map[string]ctrlruntimeclient.Client)
		createSeedImpersonationClients := make(map[string]ImpersonationClient)

		for seedName, seed := range seeds {
			cfg, err := seedKubeconfigGetter(seed)
			if err != nil {
				return nil, err
			}
			createSeedImpersonationClients[seedName] = NewImpersonationClient(cfg, mapper).CreateImpersonatedClient
			clientPrivileged, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
			if err != nil {
				return nil, err
			}
			clientsPrivileged[seedName] = clientPrivileged
		}

		return NewEtcdBackupConfigProjectProvider(
			createSeedImpersonationClients,
			clientsPrivileged,
		), nil
	}
}

func (p *EtcdBackupConfigProjectProvider) List(ctx context.Context, userInfo *provider.UserInfo, projectID string) ([]*kubermaticv1.EtcdBackupConfigList, error) {
	var etcdBackupConfigLists []*kubermaticv1.EtcdBackupConfigList
	for _, createSeedImpersonationClient := range p.createSeedImpersonatedClients {
		impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, createSeedImpersonationClient)
		if err != nil {
			return nil, err
		}

		ebcList := &kubermaticv1.EtcdBackupConfigList{}
		err = impersonationClient.List(ctx, ebcList, ctrlruntimeclient.MatchingLabels{kubermaticv1.ProjectIDLabelKey: projectID})
		if err != nil {
			return nil, err
		}
		etcdBackupConfigLists = append(etcdBackupConfigLists, ebcList)
	}

	return etcdBackupConfigLists, nil
}

func (p *EtcdBackupConfigProjectProvider) ListUnsecured(ctx context.Context, projectID string) ([]*kubermaticv1.EtcdBackupConfigList, error) {
	var etcdBackupConfigLists []*kubermaticv1.EtcdBackupConfigList
	for _, clientPrivileged := range p.clientsPrivileged {
		ebcList := &kubermaticv1.EtcdBackupConfigList{}
		err := clientPrivileged.List(ctx, ebcList, ctrlruntimeclient.MatchingLabels{kubermaticv1.ProjectIDLabelKey: projectID})
		if err != nil {
			return nil, err
		}
		etcdBackupConfigLists = append(etcdBackupConfigLists, ebcList)
	}

	return etcdBackupConfigLists, nil
}
