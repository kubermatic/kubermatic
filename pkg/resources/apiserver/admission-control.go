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

	"gopkg.in/yaml.v3"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const podNodeSelectorFileName = "podnodeselector.yaml"
const eventRateLimitFileName = "eventconfig.yaml"

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

type EventConfiguration struct {
	Kind       string       `yaml:"kind"`
	APIVersion string       `yaml:"apiVersion"`
	Limits     []EventLimit `yaml:"limits"`
}

type EventLimit struct {
	Type      string `yaml:"type"`
	QPS       int32  `yaml:"qps"`
	Burst     int32  `yaml:"burst"`
	CacheSize int32  `yaml:"cacheSize,omitempty"`
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

			if useEventRateLimitAdmissionPlugin(data) {
				eventRateLimit := AdmissionPluginConfiguration{
					Name: resources.EventRateLimitAdmissionPlugin,
					Path: fmt.Sprintf("/etc/kubernetes/adm-control/%s", eventRateLimitFileName),
				}
				admissionConfiguration.Plugins = append(admissionConfiguration.Plugins, eventRateLimit)

				eventRateLimitConfig, err := getEventRateLimitConfiguration(data)
				if err != nil {
					return nil, err
				}
				cm.Data[eventRateLimitFileName] = eventRateLimitConfig
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

func useEventRateLimitAdmissionPlugin(data *resources.TemplateData) bool {
	admissionPlugins := sets.NewString(data.Cluster().Spec.AdmissionPlugins...)
	return data.Cluster().Spec.UseEventRateLimitAdmissionPlugin || admissionPlugins.Has(resources.EventRateLimitAdmissionPlugin)
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

func getEventRateLimitConfiguration(data *resources.TemplateData) (string, error) {
	config := EventConfiguration{
		Kind:       "Configuration",
		APIVersion: "eventratelimit.admission.k8s.io/v1alpha1",
		Limits:     []EventLimit{},
	}

	if data.Cluster().Spec.EventRateLimitConfig != nil && data.Cluster().Spec.EventRateLimitConfig.Server != nil {
		limit := data.Cluster().Spec.EventRateLimitConfig.Server
		config.Limits = append(config.Limits, EventLimit{Type: "Server", QPS: limit.QPS, Burst: limit.Burst, CacheSize: limit.CacheSize})
	}

	if data.Cluster().Spec.EventRateLimitConfig != nil && data.Cluster().Spec.EventRateLimitConfig.Namespace != nil {
		limit := data.Cluster().Spec.EventRateLimitConfig.Namespace
		config.Limits = append(config.Limits, EventLimit{Type: "Namespace", QPS: limit.QPS, Burst: limit.Burst, CacheSize: limit.CacheSize})
	}

	if data.Cluster().Spec.EventRateLimitConfig != nil && data.Cluster().Spec.EventRateLimitConfig.User != nil {
		limit := data.Cluster().Spec.EventRateLimitConfig.User
		config.Limits = append(config.Limits, EventLimit{Type: "User", QPS: limit.QPS, Burst: limit.Burst, CacheSize: limit.CacheSize})
	}

	if data.Cluster().Spec.EventRateLimitConfig != nil && data.Cluster().Spec.EventRateLimitConfig.SourceAndObject != nil {
		limit := data.Cluster().Spec.EventRateLimitConfig.SourceAndObject
		config.Limits = append(config.Limits, EventLimit{Type: "SourceAndObject", QPS: limit.QPS, Burst: limit.Burst, CacheSize: limit.CacheSize})
	}

	rawConfig, err := yaml.Marshal(config)
	if err != nil {
		return "", err
	}

	return string(rawConfig), nil
}
