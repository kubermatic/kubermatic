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
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddonConfigProvider struct that holds required components of the AddonConfigProvider
type AddonConfigProvider struct {
	client            kubermaticclientset.Interface
	addonConfigLister kubermaticv1lister.AddonConfigLister
}

// NewAddonConfigProvider returns a new AddonConfigProvider
func NewAddonConfigProvider(client kubermaticclientset.Interface, addonConfigLister kubermaticv1lister.AddonConfigLister) *AddonConfigProvider {
	return &AddonConfigProvider{
		client:            client,
		addonConfigLister: addonConfigLister,
	}
}

// Get addon configuration
func (p *AddonConfigProvider) Get(addonName string) (*kubermaticv1.AddonConfig, error) {
	return p.client.KubermaticV1().AddonConfigs().Get(addonName, metav1.GetOptions{})
}

// List available addon configurations
func (p *AddonConfigProvider) List() (*kubermaticv1.AddonConfigList, error) {
	return p.client.KubermaticV1().AddonConfigs().List(metav1.ListOptions{})
}
