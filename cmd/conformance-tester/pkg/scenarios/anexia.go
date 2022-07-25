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
	anexiatypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/anexia/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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
	var scenarios []Scenario
	for _, v := range versions {
		// Flatcar
		scenarios = append(scenarios, &anexiaScenario{
			version:    v,
			datacenter: datacenter.Spec.Anexia,
			osSpec: apimodels.OperatingSystemSpec{
				Flatcar: &apimodels.FlatcarSpec{},
			},
		})
	}
	return scenarios
}

type anexiaScenario struct {
	version    *semver.Semver
	datacenter *kubermaticv1.DatacenterSpecAnexia
	osSpec     apimodels.OperatingSystemSpec
}

func (s *anexiaScenario) Name() string {
	return fmt.Sprintf("anexia-%s-%s", getOSNameFromSpec(s.osSpec), s.version.String())
}

func (s *anexiaScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
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
	// See alibaba provider for more info on this.
	return nil, errors.New("not implemented for gitops yet")

	//nolint:govet
	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderAnexia, anexiatypes.RawConfig{
		Token:      providerconfig.ConfigVarString{Value: secrets.Anexia.Token},
		TemplateID: providerconfig.ConfigVarString{Value: secrets.Anexia.TemplateID},
		VlanID:     providerconfig.ConfigVarString{Value: secrets.Anexia.VlanID},
		LocationID: providerconfig.ConfigVarString{Value: s.datacenter.LocationID},
		DiskSize:   nodeDiskSize,
		CPUs:       nodeCpu,
		Memory:     nodeMemory,
	})
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *anexiaScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}
