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

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UserProvider manages user resources.
type SettingsProvider struct {
	runtimeClient ctrlruntimeclient.Client
}

// NewUserProvider returns a user provider.
func NewSettingsProvider(runtimeClient ctrlruntimeclient.Client) *SettingsProvider {
	return &SettingsProvider{
		runtimeClient: runtimeClient,
	}
}

func (s *SettingsProvider) GetGlobalSettings() (*kubermaticv1.KubermaticSetting, error) {
	settings := &kubermaticv1.KubermaticSetting{}
	err := s.runtimeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: kubermaticv1.GlobalSettingsName}, settings)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return s.createDefaultGlobalSettings()
		}
		return nil, err
	}
	return settings, nil
}

func (s *SettingsProvider) UpdateGlobalSettings(userInfo *provider.UserInfo, settings *kubermaticv1.KubermaticSetting) (*kubermaticv1.KubermaticSetting, error) {
	if !userInfo.IsAdmin {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
	}
	if err := s.runtimeClient.Update(context.Background(), settings); err != nil {
		return nil, err
	}
	return settings, nil
}

func (s *SettingsProvider) createDefaultGlobalSettings() (*kubermaticv1.KubermaticSetting, error) {
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
			DefaultNodeCount:            10,
			DisplayDemoInfo:             false,
			DisplayAPIDocs:              false,
			DisplayTermsOfService:       false,
			EnableDashboard:             true,
			EnableOIDCKubeconfig:        false,
			UserProjectsLimit:           0,
			RestrictProjectCreation:     false,
			EnableExternalClusterImport: true,
			MachineDeploymentVMResourceQuota: kubermaticv1.MachineDeploymentVMResourceQuota{
				MinCPU:    1,
				MaxCPU:    32,
				MinRAM:    2,
				MaxRAM:    128,
				EnableGPU: false,
			},
		},
	}
	if err := s.runtimeClient.Create(context.Background(), defaultSettings); err != nil {
		return nil, err
	}
	return defaultSettings, nil
}
