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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
)

const (
	nutanixCPUs     = 2
	nutanixMemoryMB = 4096
	nutanixDiskSize = 40
)

type nutanixScenario struct {
	baseScenario
}

func (s *nutanixScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Nutanix.KKPDatacenter,
			Nutanix: &kubermaticv1.NutanixCloudSpec{
				Username: secrets.Nutanix.Username,
				Password: secrets.Nutanix.Password,
				CSI: &kubermaticv1.NutanixCSIConfig{
					Endpoint: secrets.Nutanix.CSIEndpoint,
					Password: secrets.Nutanix.CSIPassword,
					Username: secrets.Nutanix.CSIUsername,
				},
				ProxyURL:    secrets.Nutanix.ProxyURL,
				ClusterName: secrets.Nutanix.ClusterName,
				ProjectName: secrets.Nutanix.ProjectName,
			},
		},
		Version: s.version,
	}
}

func (s *nutanixScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	cloudProviderSpec := provider.NewNutanixConfig().
		WithSubnetName(secrets.Nutanix.SubnetName).
		WithCPUs(nutanixCPUs).
		WithMemoryMB(nutanixMemoryMB).
		WithDiskSize(nutanixDiskSize).
		Build()

	md, err := s.createMachineDeployment(cluster, num, cloudProviderSpec)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
