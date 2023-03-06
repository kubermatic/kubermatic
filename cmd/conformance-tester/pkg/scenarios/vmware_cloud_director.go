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
	"errors"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
)

const (
	vmwareCloudDirectorIPAllocationMode = "DHCP"
	vmwareCloudDirectorCPUs             = 2
	vmwareCloudDirectorCPUCores         = 1
	vmwareCloudDirectorMemoryMB         = 4096
	vmwareCloudDirectoDiskSize          = 20
	vmwareCloudDirectorCatalog          = "kubermatic"
	vmwareCloudDirectorStorageProfile   = "Intermediate"
)

type vmwareCloudDirectorScenario struct {
	baseScenario
}

func (s *vmwareCloudDirectorScenario) IsValid() error {
	if err := s.baseScenario.IsValid(); err != nil {
		return err
	}

	if s.operatingSystem != providerconfig.OperatingSystemUbuntu {
		return errors.New("provider only supports Flatcar")
	}

	return nil
}

func (s *vmwareCloudDirectorScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	spec := &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.VMwareCloudDirector.KKPDatacenter,
			VMwareCloudDirector: &kubermaticv1.VMwareCloudDirectorCloudSpec{
				Username:     secrets.VMwareCloudDirector.Username,
				Password:     secrets.VMwareCloudDirector.Password,
				Organization: secrets.VMwareCloudDirector.Organization,
				VDC:          secrets.VMwareCloudDirector.VDC,
				OVDCNetwork:  secrets.VMwareCloudDirector.OVDCNetwork,
				CSI: &kubermaticv1.VMwareCloudDirectorCSIConfig{
					StorageProfile: vmwareCloudDirectorStorageProfile,
				},
			},
		},
		Version: s.clusterVersion,
	}

	return spec
}

func (s *vmwareCloudDirectorScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	cloudProviderSpec := provider.NewVMwareCloudDirectorConfig().
		WithCatalog(vmwareCloudDirectorCatalog).
		WithCPUs(vmwareCloudDirectorCPUs).
		WithCPUCores(vmwareCloudDirectorCPUCores).
		WithMemoryMB(vmwareCloudDirectorMemoryMB).
		WithDiskSizeGB(vmwareCloudDirectoDiskSize).
		WithIPAllocationMode(vmwareCloudDirectorIPAllocationMode).
		Build()

	md, err := s.createMachineDeployment(cluster, num, cloudProviderSpec, sshPubKeys)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
