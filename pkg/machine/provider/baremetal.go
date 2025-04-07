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

package provider

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/machine-controller/sdk/cloudprovider/baremetal"
)

type baremetalConfig struct {
	baremetal.RawConfig
}

func NewBaremetalConfig() *baremetalConfig {
	return &baremetalConfig{}
}

func CompleteBaremetalProviderSpec(config *baremetal.RawConfig, cluster *kubermaticv1.Cluster, datacenter *kubermaticv1.DatacenterSpecBaremetal) (*baremetal.RawConfig, error) {
	if cluster != nil && cluster.Spec.Cloud.Baremetal == nil {
		return nil, fmt.Errorf("cannot use cluster to create Baremetal cloud spec as cluster uses %q", cluster.Spec.Cloud.ProviderName)
	}

	if cluster.Spec.Cloud.Baremetal.Tinkerbell != nil {
		config.Driver.Value = "tinkerbell"
	}

	return config, nil
}
