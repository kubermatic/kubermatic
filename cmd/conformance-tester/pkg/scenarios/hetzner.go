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

	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
	clusterv1alpha1 "k8c.io/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "k8c.io/machine-controller/pkg/providerconfig/types"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	hetznerServerType = "cx32"
)

type hetznerScenario struct {
	baseScenario
}

func (s *hetznerScenario) compatibleOperatingSystems() sets.Set[providerconfig.OperatingSystem] {
	return sets.New(
		providerconfig.OperatingSystemCentOS,
		providerconfig.OperatingSystemRockyLinux,
		providerconfig.OperatingSystemUbuntu,
	)
}

func (s *hetznerScenario) IsValid() error {
	if err := s.baseScenario.IsValid(); err != nil {
		return err
	}

	if compat := s.compatibleOperatingSystems(); !compat.Has(s.operatingSystem) {
		return fmt.Errorf("provider supports only %v", sets.List(compat))
	}

	return nil
}

func (s *hetznerScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Hetzner.KKPDatacenter,
			Hetzner: &kubermaticv1.HetznerCloudSpec{
				Token: secrets.Hetzner.Token,
			},
		},
		Version: s.clusterVersion,
	}
}

func (s *hetznerScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	cloudProviderSpec := provider.NewHetznerConfig().
		WithServerType(hetznerServerType).
		Build()

	md, err := s.createMachineDeployment(cluster, num, cloudProviderSpec, sshPubKeys, secrets)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
