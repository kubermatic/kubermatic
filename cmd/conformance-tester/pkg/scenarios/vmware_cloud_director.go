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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/util/sets"
)

type vmwareCloudDirectorScenario struct {
	BaseScenario
}

func (s *vmwareCloudDirectorScenario) compatibleOperatingSystems() sets.Set[providerconfig.OperatingSystem] {
	return sets.New[providerconfig.OperatingSystem](
		providerconfig.OperatingSystemUbuntu,
		providerconfig.OperatingSystemRHEL,
		providerconfig.OperatingSystemFlatcar,
		providerconfig.OperatingSystemRockyLinux,
	)
}

func (s *vmwareCloudDirectorScenario) IsValid() error {
	if err := s.BaseScenario.IsValid(); err != nil {
		return err
	}

	if compat := s.compatibleOperatingSystems(); !compat.Has(s.OperatingSystem()) {
		return fmt.Errorf("provider supports only %v", sets.List(compat))
	}

	return nil
}

func (s *vmwareCloudDirectorScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.VMwareCloudDirector.KKPDatacenter,
			VMwareCloudDirector: &kubermaticv1.VMwareCloudDirectorCloudSpec{
				Username:     secrets.VMwareCloudDirector.Username,
				Password:     secrets.VMwareCloudDirector.Password,
				Organization: secrets.VMwareCloudDirector.Organization,
				VDC:          secrets.VMwareCloudDirector.VDC,
			},
		},
		Version: s.ClusterVersion(),
	}
}

func (s *vmwareCloudDirectorScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	cloudProviderSpec := provider.NewVMwareCloudDirectorConfig().
		WithCPUs(2).
		WithCPUCores(1).
		WithMemoryMB(2048).
		WithDiskSizeGB(20).
		WithIPAllocationMode("DHCP").
		Build()

	md, err := s.CreateMachineDeployment(cluster, num, cloudProviderSpec, sshPubKeys, secrets)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
