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
	vcdtypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vmwareclouddirector/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	"k8s.io/utils/pointer"
)

const (
	vmwareCloudDirectorIPAllocationMode = "DHCP"
	vmwareCloudDirectorCPUs             = 2
	vmwareCloudDirectorCPUCores         = 1
	vmwareCloudDirectoMemoryMB          = 4096
	vmwareCloudDirectoDiskSize          = 20
	vmwareCloudDirectorCatalog          = "kubermatic"
	vmwareCloudDirectorStorageProfile   = "Intermediate"
)

// GetVMwareCloudDirectorScenarios returns a matrix of (version x operating system).
func GetVMwareCloudDirectorScenarios(versions []*semver.Semver, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*vmwareCloudDirectorScenario{
		{
			datacenter: datacenter.Spec.VMwareCloudDirector,
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
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

type vmwareCloudDirectorScenario struct {
	version          *semver.Semver
	containerRuntime string
	datacenter       *kubermaticv1.DatacenterSpecVMwareCloudDirector
	osSpec           apimodels.OperatingSystemSpec
}

func (s *vmwareCloudDirectorScenario) DeepCopy() *vmwareCloudDirectorScenario {
	version := s.version.DeepCopy()

	return &vmwareCloudDirectorScenario{
		version:          &version,
		containerRuntime: s.containerRuntime,
		osSpec:           s.osSpec,
		datacenter:       s.datacenter,
	}
}

func (s *vmwareCloudDirectorScenario) ContainerRuntime() string {
	return s.containerRuntime
}

func (s *vmwareCloudDirectorScenario) Name() string {
	return fmt.Sprintf("vmware-cloud-director-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *vmwareCloudDirectorScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	spec := &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.VMwareCloudDirector.KKPDatacenter,
					Vmwareclouddirector: &apimodels.VMwareCloudDirectorCloudSpec{
						Username:     secrets.VMwareCloudDirector.Username,
						Password:     secrets.VMwareCloudDirector.Password,
						Organization: secrets.VMwareCloudDirector.Organization,
						VDC:          secrets.VMwareCloudDirector.VDC,
						OVDCNetwork:  secrets.VMwareCloudDirector.OVDCNetwork,
						Csi: &apimodels.VMwareCloudDirectorCSIConfig{
							StorageProfile: vmwareCloudDirectorStorageProfile,
						},
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}

	return spec
}

func (s *vmwareCloudDirectorScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	spec := &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.VMwareCloudDirector.KKPDatacenter,
			VMwareCloudDirector: &kubermaticv1.VMwareCloudDirectorCloudSpec{
				Username:     secrets.VMwareCloudDirector.Username,
				Password:     secrets.VMwareCloudDirector.Password,
				Organization: secrets.VMwareCloudDirector.Organization,
				VDC:          secrets.VMwareCloudDirector.VDC,
				OVDCNetwork:  secrets.VMwareCloudDirector.OVDCNetwork,
				CSI: &kubermaticv1.VMwareCloudDirectorCSIConfig{
					StorageProfile: vmwareCloudDirectorStorageProfile,
				},
			},
		},
		Version: *s.version,
	}

	return spec
}

func (s *vmwareCloudDirectorScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.osSpec)
	replicas := int32(num)
	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Vmwareclouddirector: &apimodels.VMwareCloudDirectorNodeSpec{
							Template:         s.datacenter.Templates[osName],
							Catalog:          vmwareCloudDirectorCatalog,
							CPUs:             vmwareCloudDirectorCPUs,
							CPUCores:         vmwareCloudDirectorCPUCores,
							MemoryMB:         vmwareCloudDirectoMemoryMB,
							DiskSizeGB:       vmwareCloudDirectoDiskSize,
							IPAllocationMode: vmwareCloudDirectorIPAllocationMode,
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

func (s *vmwareCloudDirectorScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	// See alibaba provider for more info on this.
	return nil, errors.New("not implemented for gitops yet")

	//nolint:govet
	os := getOSNameFromSpec(s.osSpec)

	md, err := createMachineDeployment(num, s.version, os, s.osSpec, providerconfig.CloudProviderVMwareCloudDirector, vcdtypes.RawConfig{
		Template:         providerconfig.ConfigVarString{Value: s.datacenter.Templates[os]},
		Catalog:          providerconfig.ConfigVarString{Value: vmwareCloudDirectorCatalog},
		CPUs:             vmwareCloudDirectorCPUs,
		MemoryMB:         vmwareCloudDirectoMemoryMB,
		CPUCores:         vmwareCloudDirectorCPUCores,
		DiskSizeGB:       pointer.Int64(vmwareCloudDirectoDiskSize),
		IPAllocationMode: vmwareCloudDirectorIPAllocationMode,
	})

	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *vmwareCloudDirectorScenario) OS() apimodels.OperatingSystemSpec {
	return s.osSpec
}
