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

package scenarios

import (
	"context"
	"fmt"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/machine"
)

const (
	azureVMSize = "Standard_F2"
)

type azureScenario struct {
	baseScenario
}

func (s *azureScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Azure.KKPDatacenter,
			Azure: &kubermaticv1.AzureCloudSpec{
				ClientID:        secrets.Azure.ClientID,
				ClientSecret:    secrets.Azure.ClientSecret,
				SubscriptionID:  secrets.Azure.SubscriptionID,
				TenantID:        secrets.Azure.TenantID,
				LoadBalancerSKU: kubermaticv1.AzureStandardLBSKU,
			},
		},
		Version: s.version,
	}
}

func (s *azureScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	osSpec, err := s.OperatingSystemSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to build OS spec: %w", err)
	}

	nodeSpec := apiv1.NodeSpec{
		OperatingSystem: *osSpec,
		Cloud: apiv1.NodeCloudSpec{
			Azure: &apiv1.AzureNodeSpec{
				Size: azureVMSize,
			},
		},
	}

	config, err := machine.GetAzureProviderConfig(cluster, nodeSpec, s.datacenter)
	if err != nil {
		return nil, err
	}

	md, err := s.createMachineDeployment(num, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
