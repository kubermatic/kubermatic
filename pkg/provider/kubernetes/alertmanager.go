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
	"fmt"
	"net/http"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// AlertmanagerProvider struct that holds required components in order to manage alertmanager objects.
type AlertmanagerProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient impersonationClient
}

// NewAlertmanagerProvider returns an alertmanager provider
func NewAlertmanagerProvider(createSeedImpersonatedClient impersonationClient) *AlertmanagerProvider {
	return &AlertmanagerProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
	}
}

func AlertmanagerProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter) provider.AlertmanagerProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.AlertmanagerProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		defaultImpersonationClientForSeed := NewImpersonationClient(cfg, mapper)
		return NewAlertmanagerProvider(
			defaultImpersonationClientForSeed.CreateImpersonatedClient,
		), nil
	}
}

// Get gets an Alertmanager object and Secret which contains the configuration of this Alertmanager.
func (p *AlertmanagerProvider) Get(cluster *kubermaticv1.Cluster, userInfo *provider.UserInfo) (*kubermaticv1.Alertmanager, *corev1.Secret, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	alertmanager := &kubermaticv1.Alertmanager{}
	if err := impersonationClient.Get(ctx, types.NamespacedName{
		Name:      resources.AlertmanagerName,
		Namespace: cluster.Status.NamespaceName,
	}, alertmanager); err != nil {
		return nil, nil, err
	}
	configSecret := &corev1.Secret{}
	if err := impersonationClient.Get(ctx, types.NamespacedName{
		Name:      alertmanager.Spec.ConfigSecret.Name,
		Namespace: cluster.Status.NamespaceName,
	}, configSecret); err != nil {
		return nil, nil, err
	}
	return alertmanager, configSecret, nil
}

// Create only updates an Alertmanager object and corresponding config Secret since Alertmanager and Secret will
// be created by alertmanager configuration controller.
func (p *AlertmanagerProvider) Create(expectedAlertmanager *kubermaticv1.Alertmanager, expectedSecret *corev1.Secret, userInfo *provider.UserInfo) (*kubermaticv1.Alertmanager, *corev1.Secret, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, nil, err
	}
	ctx := context.Background()
	alertmanager := &kubermaticv1.Alertmanager{}

	if err := impersonationClient.Get(ctx, types.NamespacedName{
		Name:      expectedAlertmanager.Name,
		Namespace: expectedAlertmanager.Namespace,
	}, alertmanager); err != nil {
		return nil, nil, utilerrors.New(http.StatusInternalServerError, fmt.Errorf("failed to get alertmanager: %w", err).Error())
	}

	if alertmanager.Spec.ConfigSecret.Name == "" {
		return nil, nil, utilerrors.New(http.StatusInternalServerError, "failed to find alertmanager configuration")
	}
	secret := &corev1.Secret{}
	if err := impersonationClient.Get(ctx, types.NamespacedName{
		Name:      alertmanager.Spec.ConfigSecret.Name,
		Namespace: alertmanager.Namespace,
	}, secret); err != nil {
		return nil, nil, utilerrors.New(http.StatusInternalServerError, fmt.Errorf("failed to get config secret: %w", err).Error())
	}
	secret.Data = expectedSecret.Data
	if err := impersonationClient.Update(ctx, secret); err != nil {
		return nil, nil, err
	}
	return alertmanager, secret, nil
}

// Delete resets corresponding config Secret of Alertmanager object to the default config. Note that Delete will not remove
// Alertmanager object, it will only delete the config secret, and alertmanager controller will create default config secret.
func (p *AlertmanagerProvider) Delete(cluster *kubermaticv1.Cluster, userInfo *provider.UserInfo) error {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}
	ctx := context.Background()
	alertmanager := &kubermaticv1.Alertmanager{}
	if err := impersonationClient.Get(ctx, types.NamespacedName{
		Name:      resources.AlertmanagerName,
		Namespace: cluster.Status.NamespaceName,
	}, alertmanager); err != nil {
		return utilerrors.New(http.StatusInternalServerError, fmt.Errorf("failed to get alertmanager: %w", err).Error())
	}
	if alertmanager.Spec.ConfigSecret.Name == "" {
		return utilerrors.New(http.StatusInternalServerError, "failed to find alertmanager configuration")
	}
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertmanager.Spec.ConfigSecret.Name,
			Namespace: alertmanager.Namespace,
		},
	}
	return impersonationClient.Delete(ctx, secret)
}
