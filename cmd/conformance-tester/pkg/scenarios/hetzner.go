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
	"fmt"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	hetznertypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/hetzner/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

const (
	hetznerServerType = "cx31"
)

// GetHetznerScenarios returns a matrix of (version x operating system).
func GetHetznerScenarios(versions []*semver.Semver, _ *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*hetznerScenario{
		{
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		},
		{
			osSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
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

type hetznerScenario struct {
	version          *semver.Semver
	containerRuntime string
	osSpec           apimodels.OperatingSystemSpec
}

func (s *hetznerScenario) DeepCopy() *hetznerScenario {
	version := s.version.DeepCopy()

	return &hetznerScenario{
		version:          &version,
		containerRuntime: s.containerRuntime,
		osSpec:           s.osSpec,
	}
}

func (s *hetznerScenario) ContainerRuntime() string {
	return s.containerRuntime
}

func (s *hetznerScenario) Name() string {
	return fmt.Sprintf("hetzner-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *hetznerScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.Hetzner.KKPDatacenter,
					Hetzner: &apimodels.HetznerCloudSpec{
						Token: secrets.Hetzner.Token,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *hetznerScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Hetzner.KKPDatacenter,
			Hetzner: &kubermaticv1.HetznerCloudSpec{
				Token: secrets.Hetzner.Token,
			},
		},
		Version: *s.version,
	}
}

func (s *hetznerScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)
	nodeType := hetznerServerType

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Hetzner: &apimodels.HetznerNodeSpec{
							Type: &nodeType,
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

func (s *hetznerScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	// See alibaba provider for more info on this.
	return nil, errors.New("not implemented for gitops yet")

	//nolint:govet
	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderHetzner, hetznertypes.RawConfig{
		ServerType: providerconfig.ConfigVarString{Value: hetznerServerType},
	})
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *hetznerScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}
