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
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/machine/provider"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	nodeCpu      = 2
	nodeDiskSize = 60
	nodeMemory   = 2048
)

type anexiaScenario struct {
	baseScenario
}

func (s *anexiaScenario) compatibleOperatingSystems() sets.Set[providerconfig.OperatingSystem] {
	return sets.New[providerconfig.OperatingSystem](
		providerconfig.OperatingSystemFlatcar,
	)
}

func (s *anexiaScenario) IsValid() error {
	if err := s.baseScenario.IsValid(); err != nil {
		return err
	}

	if compat := s.compatibleOperatingSystems(); !compat.Has(s.operatingSystem) {
		return fmt.Errorf("provider supports only %v", sets.List(compat))
	}

	return nil
}

func (s *anexiaScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Anexia.KKPDatacenter,
			Anexia: &kubermaticv1.AnexiaCloudSpec{
				Token: secrets.Anexia.Token,
			},
		},
		Version: s.clusterVersion,
	}
}

func (s *anexiaScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	cloudProviderSpec := provider.NewAnexiaConfig().
		WithCPUs(nodeCpu).
		WithMemory(nodeMemory).
		AddDisk(nodeDiskSize, "ENT6").
		WithTemplateID(secrets.Anexia.TemplateID).
		WithVlanID(secrets.Anexia.VlanID).
		Build()

	md, err := s.createMachineDeployment(cluster, num, cloudProviderSpec, sshPubKeys, secrets)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
