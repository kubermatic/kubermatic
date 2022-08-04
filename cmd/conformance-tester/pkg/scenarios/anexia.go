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
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/machine"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	"k8s.io/utils/pointer"
)

const (
	nodeCpu      = 2
	nodeDiskSize = 60
	nodeMemory   = 2048
)

// GetAnexiaScenarios returns a matrix of (version x operating system).
func GetAnexiaScenarios(versions []*semver.Semver, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*anexiaScenario{
		{
			baseScenario: baseScenario{
				datacenter: datacenter,
				osSpec: apimodels.OperatingSystemSpec{
					Flatcar: &apimodels.FlatcarSpec{},
				},
			},
		},
	}

	scenarios := []Scenario{}
	for _, v := range versions {
		for _, cri := range []string{resources.ContainerRuntimeContainerd, resources.ContainerRuntimeDocker} {
			for _, scenario := range baseScenarios {
				copy := scenario.DeepCopy()
				copy.version = v
				copy.containerRuntime = cri

				scenarios = append(scenarios, copy)
			}
		}
	}

	return scenarios
}

type anexiaScenario struct {
	baseScenario
}

func (s *anexiaScenario) DeepCopy() *anexiaScenario {
	return &anexiaScenario{
		baseScenario: *s.baseScenario.DeepCopy(),
	}
}

func (s *anexiaScenario) Name() string {
	return fmt.Sprintf("anexia-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *anexiaScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.Anexia.KKPDatacenter,
					Anexia: &apimodels.AnexiaCloudSpec{
						Token: secrets.Anexia.Token,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *anexiaScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Anexia.KKPDatacenter,
			Anexia: &kubermaticv1.AnexiaCloudSpec{
				Token: secrets.Anexia.Token,
			},
		},
		Version: *s.version,
	}
}

func (s *anexiaScenario) NodeDeployments(_ context.Context, num int, secrets types.Secrets) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Anexia: &apimodels.AnexiaNodeSpec{
							DiskSize:   pointer.Int64(nodeDiskSize),
							CPUs:       pointer.Int64(nodeCpu),
							Memory:     pointer.Int64(nodeMemory),
							TemplateID: &secrets.Anexia.TemplateID,
							VlanID:     &secrets.Anexia.VlanID,
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.osSpec,
				},
			},
		},
	}, nil
}

func (s *anexiaScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	nodeSpec := apiv1.NodeSpec{
		Cloud: apiv1.NodeCloudSpec{
			Anexia: &apiv1.AnexiaNodeSpec{
				CPUs:       nodeCpu,
				Memory:     nodeMemory,
				DiskSize:   nodeDiskSize,
				TemplateID: secrets.Anexia.TemplateID,
				VlanID:     secrets.Anexia.VlanID,
			},
		},
	}

	config, err := machine.GetAnexiaProviderConfig(cluster, nodeSpec, s.datacenter)
	if err != nil {
		return nil, err
	}

	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderAnexia, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
