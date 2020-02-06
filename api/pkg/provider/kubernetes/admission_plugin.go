package kubernetes

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// admissionPluginsGetter is a function to retrieve admission plugins
type admissionPluginsGetter = func(fromVersion string) ([]string, error)

// AdmissionPluginsProvider is a object to handle admission plugins
type AdmissionPluginsProvider struct {
	admissionPluginsGetter admissionPluginsGetter
}

func NewAdmissionPluginsProvider(ctx context.Context, client ctrlruntimeclient.Client) *AdmissionPluginsProvider {
	admissionPluginsGetter := func(version string) ([]string, error) {
		if version == "" {
			return nil, fmt.Errorf("version can not be empty")
		}
		admissionPluginList := &kubermaticv1.AdmissionPluginList{}
		if err := client.List(ctx, admissionPluginList); err != nil {
			return nil, fmt.Errorf("failed to get admission plugins %v", err)
		}

		plugins := []string{}
		v, err := semver.NewVersion(version)
		if err != nil {
			return nil, err
		}
		for _, plugin := range admissionPluginList.Items {
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

	return &AdmissionPluginsProvider{admissionPluginsGetter: admissionPluginsGetter}
}

func (p *AdmissionPluginsProvider) GetAdmissionPlugins(fromVersion string) ([]string, error) {
	return p.admissionPluginsGetter(fromVersion)
}
