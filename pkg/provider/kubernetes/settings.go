/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UserProvider manages user resources.
type SettingsProvider struct {
	runtimeClient ctrlruntimeclient.Client
}

var _ provider.SettingsProvider = &SettingsProvider{}

// NewUserProvider returns a user provider.
func NewSettingsProvider(runtimeClient ctrlruntimeclient.Client) *SettingsProvider {
	return &SettingsProvider{
		runtimeClient: runtimeClient,
	}
}

func (s *SettingsProvider) GetGlobalSettings(ctx context.Context) (*kubermaticv1.KubermaticSetting, error) {
	settings := &kubermaticv1.KubermaticSetting{}
	err := s.runtimeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: kubermaticv1.GlobalSettingsName}, settings)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return s.createDefaultGlobalSettings(ctx)
		}
		return nil, err
	}
	return settings, nil
}

func (s *SettingsProvider) UpdateGlobalSettings(ctx context.Context, userInfo *provider.UserInfo, settings *kubermaticv1.KubermaticSetting) (*kubermaticv1.KubermaticSetting, error) {
	if !userInfo.IsAdmin {
		return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
	}
	if err := s.runtimeClient.Update(ctx, settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *SettingsProvider) createDefaultGlobalSettings(ctx context.Context) (*kubermaticv1.KubermaticSetting, error) {
	defaultSettings := &kubermaticv1.KubermaticSetting{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubermaticv1.GlobalSettingsName,
		},
		Spec: kubermaticv1.SettingSpec{
			CustomLinks: []kubermaticv1.CustomLink{},
			CleanupOptions: kubermaticv1.CleanupOptions{
				Enabled:  false,
				Enforced: false,
			},
			DefaultNodeCount:            2,
			DisplayDemoInfo:             false,
			DisplayAPIDocs:              false,
			DisplayTermsOfService:       false,
			EnableDashboard:             true,
			EnableOIDCKubeconfig:        false,
			UserProjectsLimit:           0,
			RestrictProjectCreation:     false,
			EnableExternalClusterImport: true,
			MachineDeploymentVMResourceQuota: &kubermaticv1.MachineFlavorFilter{
				MinCPU:    1,
				MaxCPU:    32,
				MinRAM:    2,
				MaxRAM:    128,
				EnableGPU: false,
			},
		},
	}
	if err := s.runtimeClient.Create(ctx, defaultSettings); err != nil {
		return nil, err
	}
	return defaultSettings, nil
}
