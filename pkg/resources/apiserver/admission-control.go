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

package apiserver

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const podNodeSelectorFileName = "podnodeselector.yaml"

// AdmissionConfiguration provides versioned configuration for admission controllers.
type AdmissionConfiguration struct {
	Kind string `yaml:"kind,omitempty"`

	APIVersion string `yaml:"apiVersion,omitempty"`

	// Plugins allows specifying a configuration per admission control plugin.
	Plugins []AdmissionPluginConfiguration `yaml:"plugins,omitempty"`
}

// AdmissionPluginConfiguration provides the configuration for a single plug-in.
type AdmissionPluginConfiguration struct {
	// Name is the name of the admission controller.
	// It must match the registered admission plugin name.
	Name string `yaml:"name"`

	// Path is the path to a configuration file that contains the plugin's
	// configuration
	Path string `yaml:"path"`
}

func AdmissionControlCreator(data *resources.TemplateData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.AdmissionControlConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			admissionConfiguration := AdmissionConfiguration{
				APIVersion: "apiserver.config.k8s.io/v1",
				Kind:       "AdmissionConfiguration",
				Plugins:    []AdmissionPluginConfiguration{},
			}

			deprecatedVersion, err := semver.NewSemver("1.17")
			if err != nil {
				return nil, err
			}

			// Deprecated in v1.17 in favor of apiserver.config.k8s.io/v1
			if data.Cluster().Spec.Version.LessThan(deprecatedVersion.Version) {
				admissionConfiguration.APIVersion = "apiserver.k8s.io/v1alpha1"
			}

			if usePodNodeSelectorAdmissionPlugin(data) {
				podNodeSelector := AdmissionPluginConfiguration{
					Name: resources.PodNodeSelectorAdmissionPlugin,
					Path: fmt.Sprintf("/etc/kubernetes/adm-control/%s", podNodeSelectorFileName),
				}
				admissionConfiguration.Plugins = append(admissionConfiguration.Plugins, podNodeSelector)

				podNodeConfig, err := getPodNodeSelectorAdmissionPluginConfig(data)
				if err != nil {
					return nil, err
				}
				cm.Data[podNodeSelectorFileName] = podNodeConfig
			}

			rawAdmissionConfiguration, err := yaml.Marshal(admissionConfiguration)
			if err != nil {
				return nil, err
			}

			cm.Data["admission-control.yaml"] = string(rawAdmissionConfiguration)

			return cm, nil
		}
	}
}

func usePodNodeSelectorAdmissionPlugin(data *resources.TemplateData) bool {
	admissionPlugins := sets.NewString(data.Cluster().Spec.AdmissionPlugins...)
	return data.Cluster().Spec.UsePodNodeSelectorAdmissionPlugin || admissionPlugins.Has(resources.PodNodeSelectorAdmissionPlugin)
}

func getPodNodeSelectorAdmissionPluginConfig(data *resources.TemplateData) (string, error) {
	var pluginConfig struct {
		PodNodeSelectorPluginConfig map[string]string `yaml:"podNodeSelectorPluginConfig,omitempty"`
	}

	if data.Cluster().Spec.PodNodeSelectorAdmissionPluginConfig == nil {
		data.Cluster().Spec.PodNodeSelectorAdmissionPluginConfig = map[string]string{}
	}

	pluginConfig.PodNodeSelectorPluginConfig = data.Cluster().Spec.PodNodeSelectorAdmissionPluginConfig

	rawPodNodeConfig, err := yaml.Marshal(pluginConfig)
	if err != nil {
		return "", err
	}

	return string(rawPodNodeConfig), nil
}
