package kubernetes

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// admissionPluginsGetter is a function to retrieve admission plugins
type admissionPluginsGetter = func() ([]kubermaticv1.AdmissionPlugin, error)

// AdmissionPluginsProvider is a object to handle admission plugins
type AdmissionPluginsProvider struct {
	admissionPluginsGetter admissionPluginsGetter
	client                 ctrlruntimeclient.Client
	ctx                    context.Context
}

func NewAdmissionPluginsProvider(ctx context.Context, client ctrlruntimeclient.Client) *AdmissionPluginsProvider {
	admissionPluginsGetter := func() ([]kubermaticv1.AdmissionPlugin, error) {
		admissionPluginList := &kubermaticv1.AdmissionPluginList{}
		if err := client.List(ctx, admissionPluginList); err != nil {
			return nil, fmt.Errorf("failed to get admission plugins %v", err)
		}
		return admissionPluginList.Items, nil
	}

	return &AdmissionPluginsProvider{admissionPluginsGetter: admissionPluginsGetter, client: client, ctx: ctx}
}

func (p *AdmissionPluginsProvider) ListPluginNamesFromVersion(fromVersion string) ([]string, error) {
	if fromVersion == "" {
		return nil, fmt.Errorf("fromVersion can not be empty")
	}
	admissionPluginList, err := p.admissionPluginsGetter()
	if err != nil {
		return nil, err
	}

	plugins := []string{}
	v, err := semver.NewVersion(fromVersion)
	if err != nil {
		return nil, err
	}
	for _, plugin := range admissionPluginList {
		// all without version constrain
		if plugin.Spec.FromVersion == nil {
			plugins = append(plugins, plugin.Spec.PluginName)
			continue
		}
		// version >= plugin.version
		if v.Equal(plugin.Spec.FromVersion.Version) || v.GreaterThan(plugin.Spec.FromVersion.Version) {
			plugins = append(plugins, plugin.Spec.PluginName)
		}
	}
	return plugins, nil

}

func (p *AdmissionPluginsProvider) List(userInfo *provider.UserInfo) ([]kubermaticv1.AdmissionPlugin, error) {
	if !userInfo.IsAdmin {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
	}
	admissionPluginList, err := p.admissionPluginsGetter()
	if err != nil {
		return nil, err
	}
	return admissionPluginList, nil
}

func (p *AdmissionPluginsProvider) Get(userInfo *provider.UserInfo, name string) (*kubermaticv1.AdmissionPlugin, error) {
	if !userInfo.IsAdmin {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
	}
	admissionPluginList, err := p.admissionPluginsGetter()
	if err != nil {
		return nil, err
	}
	for _, plugin := range admissionPluginList {
		if plugin.Name == name {
			return &plugin, nil
		}
	}
	return nil, kerrors.NewNotFound(schema.GroupResource{}, name)
}

func (p *AdmissionPluginsProvider) Delete(userInfo *provider.UserInfo, name string) error {

	plugin, err := p.Get(userInfo, name)
	if err != nil {
		return err
	}
	if err := p.client.Delete(p.ctx, plugin); err != nil {
		return fmt.Errorf("failed to delete admission plugins %v", err)
	}
	return nil
}

func (p *AdmissionPluginsProvider) Update(userInfo *provider.UserInfo, admissionPlugin *kubermaticv1.AdmissionPlugin) (*kubermaticv1.AdmissionPlugin, error) {
	if admissionPlugin == nil {
		return nil, fmt.Errorf("the admissionPlugin can not be nil")
	}

	oldAdmissionPlugin, err := p.Get(userInfo, admissionPlugin.Name)
	if err != nil {
		return nil, err
	}
	if err := p.client.Patch(p.ctx, admissionPlugin, ctrlruntimeclient.MergeFrom(oldAdmissionPlugin)); err != nil {
		return nil, fmt.Errorf("failed to update AdmissionPlugin: %v", err)
	}

	return admissionPlugin, nil
}
