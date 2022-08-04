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
	gcetypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/gce/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	"k8s.io/utils/pointer"
)

// GetGCPScenarios returns a matrix of (version x operating system).
func GetGCPScenarios(versions []*semver.Semver, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*gcpScenario{
		{
			datacenter: datacenter.Spec.GCP,
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		},
		{
			datacenter: datacenter.Spec.GCP,
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

type gcpScenario struct {
	version          *semver.Semver
	containerRuntime string
	datacenter       *kubermaticv1.DatacenterSpecGCP
	osSpec           apimodels.OperatingSystemSpec
}

func (s *gcpScenario) DeepCopy() *gcpScenario {
	version := s.version.DeepCopy()

	return &gcpScenario{
		version:          &version,
		containerRuntime: s.containerRuntime,
		datacenter:       s.datacenter.DeepCopy(),
		osSpec:           s.osSpec,
	}
}

func (s *gcpScenario) ContainerRuntime() string {
	return s.containerRuntime
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
	// See alibaba provider for more info on this.
	return nil, errors.New("not implemented for gitops yet")

	//nolint:govet
	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderGoogle, gcetypes.RawConfig{
		Zone:        providerconfig.ConfigVarString{Value: s.getZone()},
		MachineType: providerconfig.ConfigVarString{Value: "n1-standard-2"},
		DiskType:    providerconfig.ConfigVarString{Value: "pd-standard"},
		DiskSize:    50,
		Preemptible: providerconfig.ConfigVarBool{Value: pointer.Bool(false)},
		Labels: map[string]string{
			"kubernetes-cluster": "my-cluster",
		},
	})
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *gcpScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}

func (s *gcpScenario) getZone() string {
	return fmt.Sprintf("%s%s", s.datacenter.Region, s.datacenter.ZoneSuffixes[0])
}
