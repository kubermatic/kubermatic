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

// GetGCPScenarios returns a matrix of (version x operating system).
func GetGCPScenarios(versions []*semver.Semver, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*gcpScenario{
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

type gcpScenario struct {
	baseScenario
}

func (s *gcpScenario) DeepCopy() *gcpScenario {
	return &gcpScenario{
		baseScenario: *s.baseScenario.DeepCopy(),
	}
}

func (s *gcpScenario) Name() string {
	return fmt.Sprintf("gcp-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *gcpScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.GCP.KKPDatacenter,
					Gcp: &apimodels.GCPCloudSpec{
						ServiceAccount: secrets.GCP.ServiceAccount,
						Network:        secrets.GCP.Network,
						Subnetwork:     secrets.GCP.Subnetwork,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *gcpScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.GCP.KKPDatacenter,
			GCP: &kubermaticv1.GCPCloudSpec{
				ServiceAccount: secrets.GCP.ServiceAccount,
				Network:        secrets.GCP.Network,
				Subnetwork:     secrets.GCP.Subnetwork,
			},
		},
		Version: *s.version,
	}
}

func (s *gcpScenario) NodeDeployments(_ context.Context, num int, secrets types.Secrets) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Gcp: &apimodels.GCPNodeSpec{
							Zone:        s.getZone(),
							MachineType: "n1-standard-2",
							DiskType:    "pd-standard",
							DiskSize:    50,
							Preemptible: false,
							Labels: map[string]string{
								"kubernetes-cluster": "my-cluster",
							},
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

func (s *gcpScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	nodeSpec := apiv1.NodeSpec{
		Cloud: apiv1.NodeCloudSpec{
			GCP: &apiv1.GCPNodeSpec{
				Zone:        s.getZone(),
				MachineType: "n1-standard-2",
				DiskType:    "pd-standard",
				DiskSize:    50,
				Preemptible: false,
				Labels: map[string]string{
					"kubernetes-cluster": cluster.Name,
				},
			},
		},
	}

	config, err := machine.GetGCPProviderConfig(cluster, nodeSpec, s.datacenter)
	if err != nil {
		return nil, err
	}

	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderGoogle, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *gcpScenario) getZone() string {
	return fmt.Sprintf("%s%s", s.datacenter.Spec.GCP.Region, s.datacenter.Spec.GCP.ZoneSuffixes[0])
}
