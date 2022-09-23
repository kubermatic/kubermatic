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

package cluster

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud"
	"k8c.io/kubermatic/v2/pkg/validation"
	"k8c.io/kubermatic/v2/pkg/version"

	"k8s.io/utils/pointer"
)

// Spec builds ClusterSpec kubermatic Custom Resource from API Cluster.
// The ClusterTemplate can be nil.
func Spec(ctx context.Context, apiCluster apiv1.Cluster, template *kubermaticv1.ClusterTemplate, seed *kubermaticv1.Seed, dc *kubermaticv1.Datacenter, config *kubermaticv1.KubermaticConfiguration, secretKeyGetter provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool, features features.FeatureGate) (*kubermaticv1.ClusterSpec, error) {
	var userSSHKeysAgentEnabled = pointer.BoolPtr(true)
	if apiCluster.Spec.EnableUserSSHKeyAgent != nil {
		userSSHKeysAgentEnabled = apiCluster.Spec.EnableUserSSHKeyAgent
	}

	// Enable kubernetes-dashboard by default
	var kubernetesDashboardEnabled = true
	if apiCluster.Spec.KubernetesDashboard != nil {
		kubernetesDashboardEnabled = apiCluster.Spec.KubernetesDashboard.Enabled
	}

	// Enable OSM by default.
	var osmEnabled = pointer.BoolPtr(true)
	if apiCluster.Spec.EnableOperatingSystemManager != nil {
		osmEnabled = apiCluster.Spec.EnableOperatingSystemManager
	}

	spec := &kubermaticv1.ClusterSpec{
		HumanReadableName:                   apiCluster.Name,
		Cloud:                               apiCluster.Spec.Cloud,
		MachineNetworks:                     apiCluster.Spec.MachineNetworks,
		OIDC:                                apiCluster.Spec.OIDC,
		UpdateWindow:                        apiCluster.Spec.UpdateWindow,
		Version:                             apiCluster.Spec.Version,
		UsePodSecurityPolicyAdmissionPlugin: apiCluster.Spec.UsePodSecurityPolicyAdmissionPlugin,
		UsePodNodeSelectorAdmissionPlugin:   apiCluster.Spec.UsePodNodeSelectorAdmissionPlugin,
		UseEventRateLimitAdmissionPlugin:    apiCluster.Spec.UseEventRateLimitAdmissionPlugin,
		EnableUserSSHKeyAgent:               userSSHKeysAgentEnabled,
		EnableOperatingSystemManager:        osmEnabled,
		KubernetesDashboard: &kubermaticv1.KubernetesDashboard{
			Enabled: kubernetesDashboardEnabled,
		},
		AuditLogging:                         apiCluster.Spec.AuditLogging,
		AdmissionPlugins:                     apiCluster.Spec.AdmissionPlugins,
		OPAIntegration:                       apiCluster.Spec.OPAIntegration,
		PodNodeSelectorAdmissionPluginConfig: apiCluster.Spec.PodNodeSelectorAdmissionPluginConfig,
		EventRateLimitConfig:                 apiCluster.Spec.EventRateLimitConfig,
		ServiceAccount:                       apiCluster.Spec.ServiceAccount,
		MLA:                                  apiCluster.Spec.MLA,
		ContainerRuntime:                     apiCluster.Spec.ContainerRuntime,
		CNIPlugin:                            apiCluster.Spec.CNIPlugin,
		ExposeStrategy:                       apiCluster.Spec.ExposeStrategy,
	}

	if apiCluster.Spec.ClusterNetwork != nil {
		spec.ClusterNetwork = *apiCluster.Spec.ClusterNetwork
	}

	cloudProvider, err := CloudProviderForCluster(spec, dc, secretKeyGetter, caBundle)
	if err != nil {
		return nil, err
	}

	if err := defaulting.DefaultClusterSpec(ctx, spec, template, seed, config, cloudProvider); err != nil {
		return nil, err
	}

	versionManager := version.NewFromConfiguration(config)

	if errs := validation.ValidateNewClusterSpec(ctx, spec, dc, cloudProvider, versionManager, features, nil).ToAggregate(); errs != nil {
		return spec, errs
	}

	return spec, nil
}

func CloudProviderForCluster(spec *kubermaticv1.ClusterSpec, dc *kubermaticv1.Datacenter, secretKeyGetter provider.SecretKeySelectorValueFunc, caBundle *x509.CertPool) (provider.CloudProvider, error) {
	providerName, err := provider.ClusterCloudProviderName(spec.Cloud)
	if err != nil {
		return nil, fmt.Errorf("invalid cloud spec: %w", err)
	}
	if providerName == "" {
		return nil, errors.New("cluster has no cloud provider")
	}

	return cloud.Provider(dc, secretKeyGetter, caBundle)
}
