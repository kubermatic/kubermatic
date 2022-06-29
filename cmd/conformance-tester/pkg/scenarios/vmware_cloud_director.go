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
	"strings"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	vcdtypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vmwareclouddirector/types"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	"k8s.io/utils/pointer"
)

const (
	vmwareCloudDirectorDatacenter       = "vmware-cloud-director-ger"
	vmwareCloudDirectorIPAllocationMode = "DHCP"
	vmwareCloudDirectorCPUs             = 2
	vmwareCloudDirectorCPUCores         = 1
	vmwareCloudDirectoMemoryMB          = 4096
	vmwareCloudDirectoDiskSize          = 20
	vmwareCloudDirectorCatalog          = "kubermatic"
	vmwareCloudDirectorStorageProfile   = "Intermediate"
)

// GetVMwareCloudDirectorScenarios returns a matrix of (version x operating system).
func GetVMwareCloudDirectorScenarios(versions []*semver.Semver) []Scenario {
	var scenarios []Scenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &vmwareCloudDirectorScenario{
			version: v,
			osSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
	}

	return scenarios
}

type vmwareCloudDirectorScenario struct {
	version *semver.Semver
	osSpec  apimodels.OperatingSystemSpec
}

func (s *vmwareCloudDirectorScenario) Name() string {
	return fmt.Sprintf("vmware-cloud-director-%s-%s", getOSNameFromSpec(s.osSpec), strings.ReplaceAll(s.version.String(), ".", "-"))
}

func (s *vmwareCloudDirectorScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	spec := &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: vmwareCloudDirectorDatacenter,
					VmwareCloudDirector: &apimodels.VMwareCloudDirectorCloudSpec{
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
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: vmwareCloudDirectorDatacenter,
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
						VmwareCloudDirector: &apimodels.VMwareCloudDirectorNodeSpec{
							Template:         fmt.Sprintf("machine-controller-%s", osName),
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
	template := fmt.Sprintf("machine-controller-%s", os)

	md, err := createMachineDeployment(num, s.version, os, s.osSpec, providerconfig.CloudProviderVMwareCloudDirector, vcdtypes.RawConfig{
		Template:         providerconfig.ConfigVarString{Value: template},
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
