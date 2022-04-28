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
	kubenetutil "k8s.io/apimachinery/pkg/util/net"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// EtcdRestoreProvider struct that holds required components in order manage etcd backup configs.
type EtcdRestoreProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient ImpersonationClient
	clientPrivileged             ctrlruntimeclient.Client
}

var _ provider.EtcdRestoreProvider = &EtcdRestoreProvider{}
var _ provider.PrivilegedEtcdRestoreProvider = &EtcdRestoreProvider{}

// NewEtcdRestoreProvider returns a etcd restore provider.
func NewEtcdRestoreProvider(createSeedImpersonatedClient ImpersonationClient, client ctrlruntimeclient.Client) *EtcdRestoreProvider {
	return &EtcdRestoreProvider{
		clientPrivileged:             client,
		createSeedImpersonatedClient: createSeedImpersonatedClient,
	}
}

func EtcdRestoreProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter) provider.EtcdRestoreProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.EtcdRestoreProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		defaultImpersonationClientForSeed := NewImpersonationClient(cfg, mapper)
		clientPrivileged, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
		if err != nil {
			return nil, err
		}
		return NewEtcdRestoreProvider(
			defaultImpersonationClientForSeed.CreateImpersonatedClient,
			clientPrivileged,
		), nil
	}
}

func (p *EtcdRestoreProvider) Create(ctx context.Context, userInfo *provider.UserInfo, etcdRestore *kubermaticv1.EtcdRestore) (*kubermaticv1.EtcdRestore, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	err = impersonationClient.Create(ctx, etcdRestore)
	return etcdRestore, err
}

func (p *EtcdRestoreProvider) CreateUnsecured(ctx context.Context, etcdRestore *kubermaticv1.EtcdRestore) (*kubermaticv1.EtcdRestore, error) {
	err := p.clientPrivileged.Create(ctx, etcdRestore)
	return etcdRestore, err
}

func (p *EtcdRestoreProvider) Get(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdRestore, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	er := &kubermaticv1.EtcdRestore{}
	err = impersonationClient.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Status.NamespaceName}, er)
	return er, err
}

func (p *EtcdRestoreProvider) GetUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdRestore, error) {
	er := &kubermaticv1.EtcdRestore{}
	err := p.clientPrivileged.Get(ctx, types.NamespacedName{Name: name, Namespace: cluster.Status.NamespaceName}, er)
	return er, err
}

func (p *EtcdRestoreProvider) List(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdRestoreList, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	erList := &kubermaticv1.EtcdRestoreList{}
	err = impersonationClient.List(ctx, erList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName))
	return erList, err
}

func (p *EtcdRestoreProvider) ListUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdRestoreList, error) {
	erList := &kubermaticv1.EtcdRestoreList{}
	err := p.clientPrivileged.List(ctx, erList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName))
	return erList, err
}

func (p *EtcdRestoreProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) error {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}

	er := &kubermaticv1.EtcdRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	return impersonationClient.Delete(ctx, er)
}

func (p *EtcdRestoreProvider) DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster, name string) error {
	er := &kubermaticv1.EtcdRestore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cluster.Status.NamespaceName,
		},
	}
	return p.clientPrivileged.Delete(ctx, er)
}

// EtcdRestoreProjectProvider struct that holds required components in order manage etcd backup restores across projects.
type EtcdRestoreProjectProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClients map[string]ImpersonationClient
	clientsPrivileged             map[string]ctrlruntimeclient.Client
}

var _ provider.EtcdRestoreProjectProvider = &EtcdRestoreProjectProvider{}
var _ provider.PrivilegedEtcdRestoreProjectProvider = &EtcdRestoreProjectProvider{}

// NewEtcdRestoreProjectProvider returns an etcd restore global provider.
func NewEtcdRestoreProjectProvider(createSeedImpersonatedClients map[string]ImpersonationClient, clients map[string]ctrlruntimeclient.Client) *EtcdRestoreProjectProvider {
	return &EtcdRestoreProjectProvider{
		clientsPrivileged:             clients,
		createSeedImpersonatedClients: createSeedImpersonatedClients,
	}
}

func EtcdRestoreProjectProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter) provider.EtcdRestoreProjectProviderGetter {
	return func(seeds map[string]*kubermaticv1.Seed) (provider.EtcdRestoreProjectProvider, error) {
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

		return NewEtcdRestoreProjectProvider(
			createSeedImpersonationClients,
			clientsPrivileged,
		), nil
	}
}

func (p *EtcdRestoreProjectProvider) List(ctx context.Context, userInfo *provider.UserInfo, projectID string) ([]*kubermaticv1.EtcdRestoreList, error) {
	var etcdRestoreLists []*kubermaticv1.EtcdRestoreList
	for _, createSeedImpersonationClient := range p.createSeedImpersonatedClients {
		impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, createSeedImpersonationClient)
		if err != nil {
			return nil, err
		}

		erList := &kubermaticv1.EtcdRestoreList{}
		err = impersonationClient.List(ctx, erList, ctrlruntimeclient.MatchingLabels{kubermaticv1.ProjectIDLabelKey: projectID})
		if err != nil {
			// skip if cluster is unreachable
			if kubenetutil.IsConnectionRefused(err) {
				continue
			}
			return nil, err
		}
		etcdRestoreLists = append(etcdRestoreLists, erList)
	}

	return etcdRestoreLists, nil
}

func (p *EtcdRestoreProjectProvider) ListUnsecured(ctx context.Context, projectID string) ([]*kubermaticv1.EtcdRestoreList, error) {
	var etcdRestoreLists []*kubermaticv1.EtcdRestoreList
	for _, clientPrivileged := range p.clientsPrivileged {
		erList := &kubermaticv1.EtcdRestoreList{}
		err := clientPrivileged.List(ctx, erList, ctrlruntimeclient.MatchingLabels{kubermaticv1.ProjectIDLabelKey: projectID})
		if err != nil {
			// skip if cluster is unreachable
			if kubenetutil.IsConnectionRefused(err) {
				continue
			}
			return nil, err
		}
		etcdRestoreLists = append(etcdRestoreLists, erList)
	}

	return etcdRestoreLists, nil
}
