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
	"fmt"
	"k8s.io/apimachinery/pkg/watch"

	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// UserProvider manages user resources
type SettingsProvider struct {
	client         kubermaticclientset.Interface
	settingsLister kubermaticv1lister.KubermaticSettingLister
}

// NewUserProvider returns a user provider
func NewSettingsProvider(client kubermaticclientset.Interface, settingsLister kubermaticv1lister.KubermaticSettingLister) *SettingsProvider {
	return &SettingsProvider{
		client:         client,
		settingsLister: settingsLister,
	}
}

func (s *SettingsProvider) GetGlobalSettings() (*kubermaticv1.KubermaticSetting, error) {
	settings, err := s.settingsLister.Get(kubermaticv1.GlobalSettingsName)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return s.createDefaultGlobalSettings()
		}
		return nil, err
	}
	return settings, nil
}

func (s *SettingsProvider) WatchGlobalSettings() (watch.Interface, error) {
	return s.client.KubermaticV1().KubermaticSettings().Watch(v1.ListOptions{})
}

func (s *SettingsProvider) UpdateGlobalSettings(userInfo *provider.UserInfo, settings *kubermaticv1.KubermaticSetting) (*kubermaticv1.KubermaticSetting, error) {
	if !userInfo.IsAdmin {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
	}
	return s.client.KubermaticV1().KubermaticSettings().Update(settings)
}

func (s *SettingsProvider) createDefaultGlobalSettings() (*kubermaticv1.KubermaticSetting, error) {
	defaultSettings := &kubermaticv1.KubermaticSetting{
		ObjectMeta: v1.ObjectMeta{
			Name: kubermaticv1.GlobalSettingsName,
		},
		Spec: kubermaticv1.SettingSpec{
			CustomLinks: []kubermaticv1.CustomLink{},
			CleanupOptions: kubermaticv1.CleanupOptions{
				Enabled:  false,
				Enforced: false,
			},
			DefaultNodeCount:      10,
			ClusterTypeOptions:    kubermaticv1.ClusterTypeAll,
			DisplayDemoInfo:       false,
			DisplayAPIDocs:        false,
			DisplayTermsOfService: false,
			EnableDashboard:       true,
			EnableOIDCKubeconfig:  false,
		},
	}
	return s.client.KubermaticV1().KubermaticSettings().Create(defaultSettings)
}
