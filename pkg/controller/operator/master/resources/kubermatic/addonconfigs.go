/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package kubermatic

import (
	_ "embed"
	"encoding/base64"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling/modifier"
)

var (
	//go:embed static/kube-state-metrics.svg
	kubeStateMetricsLogo []byte

	//go:embed static/kubeflow.svg
	kubeflowLogo []byte

	//go:embed static/kubevirt.svg
	kubeVirtLogo []byte

	//go:embed static/multus.svg
	multusLogo []byte

	//go:embed static/node-exporter.svg
	nodeExporterLogo []byte
)

const (
	svgFormat = "svg+xml"
	pngFormat = "png"
)

func AddonConfigsReconcilers() []reconciling.NamedAddonConfigReconcilerFactory {
	return []reconciling.NamedAddonConfigReconcilerFactory{
		makeReconciler(kubeStateMetricsAddonConfig),
		makeReconciler(kubeVirtAddonConfig),
		makeReconciler(kubeflowAddonConfig),
		makeReconciler(multusAddonConfig),
		makeReconciler(nodeExporterAddonConfig),
	}
}

func makeReconciler(addonConfigReconciler func() *kubermaticv1.AddonConfig) reconciling.NamedAddonConfigReconcilerFactory {
	addonConfig := addonConfigReconciler()

	return func() (name string, create reconciling.AddonConfigReconciler) {
		return addonConfig.Name, func(existing *kubermaticv1.AddonConfig) (*kubermaticv1.AddonConfig, error) {
			// determine whether the config was created (and is therefore managed) by KKP
			// (this is the default for KKP 2.22+) or whether an admin has manually created it.
			owned := false

			label, exists := existing.Labels[modifier.ManagedByLabel]
			if label == common.OperatorName {
				owned = true
			} else if !exists {
				// if the AddonConfig doesn't exist yet, we assume ownership over it
				owned = existing.ResourceVersion == ""
			}

			// do not touch AddonConfigs managed by someone else
			if !owned {
				return existing, nil
			}

			existing.Spec = addonConfig.Spec

			// set the ownership label here instead of using the ownershid modifier, as we
			// only want to set the ownership conditionally
			if existing.Labels == nil {
				existing.Labels = map[string]string{}
			}
			existing.Labels[modifier.ManagedByLabel] = common.OperatorName

			return existing, nil
		}
	}
}

func base64Encode(logo []byte) string {
	return base64.StdEncoding.EncodeToString(logo)
}

func kubeStateMetricsAddonConfig() *kubermaticv1.AddonConfig {
	config := &kubermaticv1.AddonConfig{}
	config.Name = "kube-state-metrics"
	config.Spec.Description = "kube-state-metrics is an agent to generate and expose cluster-level metrics of Kubernetes API objects in Prometheus format."
	config.Spec.ShortDescription = "kube-state-metrics exposes cluster-level metrics."
	config.Spec.Logo = base64Encode(kubeStateMetricsLogo)
	config.Spec.LogoFormat = svgFormat

	return config
}

func kubeflowAddonConfig() *kubermaticv1.AddonConfig {
	config := &kubermaticv1.AddonConfig{}
	config.Name = "kubeflow"
	config.Spec.Description = "Kubeflow machine learning toolkit for Kubernetes"
	config.Spec.ShortDescription = "Kubeflow machine learning toolkit for Kubernetes"
	config.Spec.Logo = base64Encode(kubeflowLogo)
	config.Spec.LogoFormat = svgFormat
	config.Spec.Controls = []kubermaticv1.AddonFormControl{
		{
			DisplayName:  "Expose via LoadBalancer",
			HelpText:     "The Kubeflow dashboard will be exposed via a LoadBalancer service instead of a NodePort service.",
			InternalName: "ExposeLoadBalancer",
			Type:         "boolean",
		},
		{
			DisplayName:  "Enable TLS",
			HelpText:     "TLS will be enabled and a certificate will be automatically issued for the specified domain name.",
			InternalName: "EnableTLS",
			Type:         "boolean",
		},
		{
			DisplayName:  "Expose via LoadBalancer",
			HelpText:     "NVIDIA GPU Operator will be installed. Also installs Node Feature Discovery for Kubernetes.",
			InternalName: "Install NVIDIA GPU Operator",
			Type:         "boolean",
		},
		{
			DisplayName:  "Install AMD GPU Device Plugin",
			HelpText:     "AMD GPU Device Plugin will be installed. Also installs AMG GPU Node Labeler.",
			InternalName: "AMDDevicePlugin",
			Type:         "boolean",
		},
		{
			DisplayName:  "Domain Name",
			HelpText:     "Domain name for accessing the Kubeflow dashboard. Make sure to set up your DNS accordingly.",
			InternalName: "DomainName",
			Type:         "text",
		},
		{
			DisplayName:  "OIDC Provider URL",
			HelpText:     "URL of external OIDC provider, e.g. the Kubermatic Dex instance. If not provided, static users will be used.",
			InternalName: "OIDCProviderURL",
			Type:         "text",
		},
		{
			DisplayName:  "OIDC Secret",
			HelpText:     "Secret string shared between the OIDC provider and Kubeflow. If not provided, the default one will be used.",
			InternalName: "OIDCSecret",
			Type:         "text",
		},
		{
			DisplayName:  "Enable Istio RBAC",
			HelpText:     "Enable Istio RBAC (Role Based Access Control) for multi-tenancy.",
			InternalName: "EnableIstioRBAC",
			Type:         "boolean",
		},
	}

	return config
}

func kubeVirtAddonConfig() *kubermaticv1.AddonConfig {
	config := &kubermaticv1.AddonConfig{}
	config.Name = "kubevirt"
	config.Spec.Description = "Virtual Machine management on Kubernetes"
	config.Spec.ShortDescription = "Virtual Machine management on Kubernetes"
	config.Spec.Logo = base64Encode(kubeVirtLogo)
	config.Spec.LogoFormat = svgFormat
	config.Spec.Controls = []kubermaticv1.AddonFormControl{
		{
			DisplayName:  "KubeVirt configuration",
			HelpText:     "Configuration that allows you to enable developer settings, feature gates etc., take a look at KubeVirt project for more details.",
			InternalName: "Configuration",
			Type:         "text-area",
		},
	}

	return config
}

func multusAddonConfig() *kubermaticv1.AddonConfig {
	config := &kubermaticv1.AddonConfig{}
	config.Name = "multus"
	config.Spec.Description = "The Multus CNI allows pods to have multiple interfaces and using features like SR-IOV."
	config.Spec.ShortDescription = "Multus CNI"
	config.Spec.Logo = base64Encode(multusLogo)
	config.Spec.LogoFormat = svgFormat

	return config
}

func nodeExporterAddonConfig() *kubermaticv1.AddonConfig {
	config := &kubermaticv1.AddonConfig{}
	config.Name = "node-exporter"
	config.Spec.Description = "The Prometheus Node Exporter exposes a wide variety of hardware- and kernel-related metrics."
	config.Spec.ShortDescription = "The exporter for machine metrics."
	config.Spec.Logo = base64Encode(nodeExporterLogo)
	config.Spec.LogoFormat = svgFormat

	return config
}
