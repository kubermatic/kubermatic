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
	alibabatypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/alibaba/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

const (
	alibabaDatacenter = "alibaba-eu-central-1a"
)

// GetAlibabaScenarios returns a matrix of (version x operating system).
func GetAlibabaScenarios(versions []*semver.Semver) []Scenario {
	var scenarios []Scenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &alibabaScenario{
			version: v,
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CentOS
		scenarios = append(scenarios, &alibabaScenario{
			version: v,
			osSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type alibabaScenario struct {
	version *semver.Semver
	osSpec  apimodels.OperatingSystemSpec
}

func (s *alibabaScenario) Name() string {
	return fmt.Sprintf("alibaba-%s-%s", getOSNameFromSpec(s.osSpec), s.version.String())
}

func (s *alibabaScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: alibabaDatacenter,
					Alibaba: &apimodels.AlibabaCloudSpec{
						AccessKeySecret: secrets.Alibaba.AccessKeySecret,
						AccessKeyID:     secrets.Alibaba.AccessKeyID,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *alibabaScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: alibabaDatacenter,
			Alibaba: &kubermaticv1.AlibabaCloudSpec{
				AccessKeySecret: secrets.Alibaba.AccessKeySecret,
				AccessKeyID:     secrets.Alibaba.AccessKeyID,
			},
		},
		Version: *s.version,
	}
}

func (s *alibabaScenario) NodeDeployments(_ context.Context, num int, secrets types.Secrets, _ *kubermaticv1.Datacenter) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Alibaba: &apimodels.AlibabaNodeSpec{
							InstanceType:            "ecs.c6.xsmall",
							DiskSize:                "40",
							DiskType:                "cloud_efficiency",
							VSwitchID:               "vsw-gw8g8mn4ohmj483hsylmn",
							InternetMaxBandwidthOut: "10",
							ZoneID:                  alibabaDatacenter,
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

func (s *alibabaScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, _ *kubermaticv1.Datacenter) ([]clusterv1alpha1.MachineDeployment, error) {
	// It is unfortunately not possible to convert an apimodels.NodeDeployment into anything
	// useful for GitOps, so this function has to completely reimplement everything (most of
	// the logic from pkg/resources/machine/common.go); since #9541 focused only on AWS, the
	// following code is most likely incomplete and to ensure that nobody gets tripped up by
	// that, we instead return a very clear error message right away. We leave the code stub
	// behind so that when someone wants to implement this specific scenario for GitOps, they
	// have a starting point.
	return nil, errors.New("not implemented for gitops yet")

	//nolint:govet
	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderAlibaba, alibabatypes.RawConfig{
		InstanceType:            providerconfig.ConfigVarString{Value: "ecs.c6.xsmall"},
		DiskSize:                providerconfig.ConfigVarString{Value: "40"},
		DiskType:                providerconfig.ConfigVarString{Value: "cloud_efficiency"},
		VSwitchID:               providerconfig.ConfigVarString{Value: "vsw-gw8g8mn4ohmj483hsylmn"},
		InternetMaxBandwidthOut: providerconfig.ConfigVarString{Value: "10"},
		ZoneID:                  providerconfig.ConfigVarString{Value: alibabaDatacenter},
	})
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *alibabaScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}
