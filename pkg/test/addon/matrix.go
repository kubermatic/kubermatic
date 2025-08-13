/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package addon

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

var (
	csiProviders = []kubermaticv1.ProviderType{
		kubermaticv1.AWSCloudProvider,
		kubermaticv1.AzureCloudProvider,
		kubermaticv1.DigitaloceanCloudProvider,
		kubermaticv1.GCPCloudProvider,
		kubermaticv1.HetznerCloudProvider,
		kubermaticv1.KubevirtCloudProvider,
		kubermaticv1.NutanixCloudProvider,
		kubermaticv1.OpenstackCloudProvider,
		kubermaticv1.VSphereCloudProvider,
		kubermaticv1.VMwareCloudDirectorCloudProvider,
	}

	// anyProvider is used when we want to test against, but do not care against which provider.
	anyProvider = []kubermaticv1.ProviderType{kubermaticv1.AWSCloudProvider}

	AddonProviderMatrix = map[string][]kubermaticv1.ProviderType{
		"aws-node-termination-handler": {kubermaticv1.AWSCloudProvider},
		"azure-cloud-node-manager":     {kubermaticv1.AzureCloudProvider},
		"canal":                        anyProvider,
		"cilium":                       anyProvider,
		"csi":                          csiProviders,
		"default-storage-class":        csiProviders,
		"hubble":                       anyProvider,
		"kube-proxy":                   anyProvider,
		"kube-state-metrics":           anyProvider,
		"kubeadm-configmap":            anyProvider,
		"kubelet-configmap":            anyProvider,
		"metallb":                      anyProvider,
		"multus":                       anyProvider,
		"node-exporter":                anyProvider,
		"openvpn":                      anyProvider,
		"pod-security-policy":          anyProvider,
		"rbac":                         anyProvider,
	}

	// Some addons rely on the manifests provided by another addon. This map
	// contains these dependencies and is evaluated recursively. Addons not listed
	// here have no dependencies.
	RequiredAddons = map[string][]string{
		// default-storage-class relies on the CSI addon installing the snapshot CRDs
		"default-storage-class": {"csi"},
	}
)
