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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// BackupCredentialsProvider struct that holds required components in order manage backup credentials.
type BackupCredentialsProvider struct {
	clientPrivileged ctrlruntimeclient.Client
}

var _ provider.BackupCredentialsProvider = &BackupCredentialsProvider{}

// NewBackupCredentialsProvider returns a  backup credential provider.
func NewBackupCredentialsProvider(client ctrlruntimeclient.Client) *BackupCredentialsProvider {
	return &BackupCredentialsProvider{
		clientPrivileged: client,
	}
}

func BackupCredentialsProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter) provider.BackupCredentialsProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.BackupCredentialsProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
		if err != nil {
			return nil, err
		}
		return NewBackupCredentialsProvider(client), nil
	}
}

func (p *BackupCredentialsProvider) CreateUnsecured(ctx context.Context, credentials *corev1.Secret) (*corev1.Secret, error) {
	err := p.clientPrivileged.Create(ctx, credentials)
	return credentials, err
}

func (p *BackupCredentialsProvider) GetUnsecured(ctx context.Context, credentialName string) (*corev1.Secret, error) {
	credentials := &corev1.Secret{}
	err := p.clientPrivileged.Get(ctx, types.NamespacedName{
		Name:      credentialName,
		Namespace: metav1.NamespaceSystem,
	}, credentials)
	return credentials, err
}

func (p *BackupCredentialsProvider) UpdateUnsecured(ctx context.Context, newSecret *corev1.Secret) (*corev1.Secret, error) {
	err := p.clientPrivileged.Update(ctx, newSecret)
	return newSecret, err
}
