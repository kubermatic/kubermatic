package kubernetes

import (
	"context"
	"fmt"

	kubermaticclientset "github.com/kubermatic/kubermatic/pkg/crd/client/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UserProvider manages user resources
type SettingsProvider struct {
	client        kubermaticclientset.Interface
	runtimeClient ctrlruntimeclient.Client
}

// NewUserProvider returns a user provider
func NewSettingsProvider(client kubermaticclientset.Interface, runtimeClient ctrlruntimeclient.Client) *SettingsProvider {
	return &SettingsProvider{
		client:        client,
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

func (s *SettingsProvider) WatchGlobalSettings() (watch.Interface, error) {
	return s.client.KubermaticV1().KubermaticSettings().Watch(v1.ListOptions{})
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
			ClusterTypeOptions:    kubermaticv1.ClusterTypeKubernetes,
			DisplayDemoInfo:       false,
			DisplayAPIDocs:        false,
			DisplayTermsOfService: false,
			EnableDashboard:       true,
			EnableOIDCKubeconfig:  false,
		},
	}
	if err := s.runtimeClient.Create(context.Background(), defaultSettings); err != nil {
		return nil, err
	}
	return defaultSettings, nil
}
