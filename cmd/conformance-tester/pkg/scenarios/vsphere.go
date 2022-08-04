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

// GetVSphereScenarios returns a matrix of (version x operating system).
func GetVSphereScenarios(scenarioOptions []string, versions []*semver.Semver, datacenter *kubermaticv1.Datacenter) []Scenario {
	var (
		customFolder     bool
		datastoreCluster bool
	)

	for _, opt := range scenarioOptions {
		switch opt {
		case "custom-folder":
			customFolder = true
		case "datastore-cluster":
			datastoreCluster = true
		}
	}

	baseScenarios := []*vSphereScenario{
		{
			baseScenario: baseScenario{
				datacenter: datacenter,
				osSpec: apimodels.OperatingSystemSpec{
					Ubuntu: &apimodels.UbuntuSpec{},
				},
			},
			customFolder:     customFolder,
			datastoreCluster: datastoreCluster,
		},
		{
			baseScenario: baseScenario{
				datacenter: datacenter,
				osSpec: apimodels.OperatingSystemSpec{
					Centos: &apimodels.CentOSSpec{},
				},
			},
			customFolder:     customFolder,
			datastoreCluster: datastoreCluster,
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

type vSphereScenario struct {
	baseScenario

	customFolder     bool
	datastoreCluster bool
}

func (s *vSphereScenario) DeepCopy() *vSphereScenario {
	return &vSphereScenario{
		baseScenario:     *s.baseScenario.DeepCopy(),
		customFolder:     s.customFolder,
		datastoreCluster: s.datastoreCluster,
	}
}

func (s *vSphereScenario) Name() string {
	return fmt.Sprintf("vsphere-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *vSphereScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	spec := &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.VSphere.KKPDatacenter,
					Vsphere: &apimodels.VSphereCloudSpec{
						Username:  secrets.VSphere.Username,
						Password:  secrets.VSphere.Password,
						Datastore: s.datacenter.Spec.VSphere.DefaultDatastore,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}

	if s.customFolder {
		spec.Cluster.Spec.Cloud.Vsphere.Folder = fmt.Sprintf("%s/custom_folder_test", s.datacenter.Spec.VSphere.RootPath)
	}

	if s.datastoreCluster {
		spec.Cluster.Spec.Cloud.Vsphere.DatastoreCluster = "dsc-1"
		spec.Cluster.Spec.Cloud.Vsphere.Datastore = ""
	}

	return spec
}

func (s *vSphereScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	spec := &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.VSphere.KKPDatacenter,
			VSphere: &kubermaticv1.VSphereCloudSpec{
				Username:  secrets.VSphere.Username,
				Password:  secrets.VSphere.Password,
				Datastore: s.datacenter.Spec.VSphere.DefaultDatastore,
			},
		},
		Version: *s.version,
	}

	if s.customFolder {
		spec.Cloud.VSphere.Folder = fmt.Sprintf("%s/custom_folder_test", s.datacenter.Spec.VSphere.RootPath)
	}

	if s.datastoreCluster {
		spec.Cloud.VSphere.DatastoreCluster = "dsc-1"
		spec.Cloud.VSphere.Datastore = ""
	}

	return spec
}

func (s *vSphereScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.osSpec)
	replicas := int32(num)
	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Vsphere: &apimodels.VSphereNodeSpec{
							Template: s.datacenter.Spec.VSphere.Templates[osName],
							CPUs:     2,
							Memory:   4096,
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

func (s *vSphereScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	os := getOSNameFromSpec(s.osSpec)
	nodeSpec := apiv1.NodeSpec{
		Cloud: apiv1.NodeCloudSpec{
			VSphere: &apiv1.VSphereNodeSpec{
				CPUs:       2,
				Memory:     4096,
				DiskSizeGB: pointer.Int64(10),
				Template:   s.datacenter.Spec.VSphere.Templates[os],
			},
		},
	}

	config, err := machine.GetVSphereProviderConfig(cluster, nodeSpec, s.datacenter)
	if err != nil {
		return nil, err
	}

	md, err := createMachineDeployment(num, s.version, os, s.osSpec, providerconfig.CloudProviderVsphere, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}
