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
)

const (
	openStackFlavor                    = "m1.small"
	openStackFloatingIPPool            = "ext-net"
	openStackInstanceReadyCheckPeriod  = "5s"
	openStackInstanceReadyCheckTimeout = "2m"
)

// GetOpenStackScenarios returns a matrix of (version x operating system).
func GetOpenStackScenarios(versions []*semver.Semver, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*openStackScenario{
		{
			baseScenario: baseScenario{
				datacenter: datacenter,
				osSpec: apimodels.OperatingSystemSpec{
					Ubuntu: &apimodels.UbuntuSpec{},
				},
			},
		},
		{
			baseScenario: baseScenario{
				datacenter: datacenter,
				osSpec: apimodels.OperatingSystemSpec{
					Centos: &apimodels.CentOSSpec{},
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

type openStackScenario struct {
	baseScenario
}

func (s *openStackScenario) DeepCopy() *openStackScenario {
	return &openStackScenario{
		baseScenario: *s.baseScenario.DeepCopy(),
	}
}

func (s *openStackScenario) Name() string {
	return fmt.Sprintf("openstack-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *openStackScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.OpenStack.KKPDatacenter,
					Openstack: &apimodels.OpenstackCloudSpec{
						Domain:         secrets.OpenStack.Domain,
						Project:        secrets.OpenStack.Project,
						ProjectID:      secrets.OpenStack.ProjectID,
						Username:       secrets.OpenStack.Username,
						Password:       secrets.OpenStack.Password,
						FloatingIPPool: openStackFloatingIPPool,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *openStackScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.OpenStack.KKPDatacenter,
			Openstack: &kubermaticv1.OpenstackCloudSpec{
				Domain:         secrets.OpenStack.Domain,
				Project:        secrets.OpenStack.Project,
				ProjectID:      secrets.OpenStack.ProjectID,
				Username:       secrets.OpenStack.Username,
				Password:       secrets.OpenStack.Password,
				FloatingIPPool: openStackFloatingIPPool,
			},
		},
		Version: *s.version,
	}
}

func (s *openStackScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.osSpec)
	flavor := openStackFlavor
	image := s.datacenter.Spec.Openstack.Images[osName]
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Openstack: &apimodels.OpenstackNodeSpec{
							Flavor:                    &flavor,
							Image:                     &image,
							InstanceReadyCheckPeriod:  openStackInstanceReadyCheckPeriod,
							InstanceReadyCheckTimeout: openStackInstanceReadyCheckTimeout,
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

func (s *openStackScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	os := getOSNameFromSpec(s.osSpec)

	nodeSpec := apiv1.NodeSpec{
		Cloud: apiv1.NodeCloudSpec{
			Openstack: &apiv1.OpenstackNodeSpec{
				Flavor:                    openStackFlavor,
				Image:                     s.datacenter.Spec.Openstack.Images[os],
				InstanceReadyCheckPeriod:  openStackInstanceReadyCheckPeriod,
				InstanceReadyCheckTimeout: openStackInstanceReadyCheckTimeout,
			},
		},
	}

	config, err := machine.GetOpenstackProviderConfig(cluster, nodeSpec, s.datacenter)
	if err != nil {
		return nil, err
	}

	md, err := createMachineDeployment(num, s.version, os, s.osSpec, providerconfig.CloudProviderOpenstack, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
