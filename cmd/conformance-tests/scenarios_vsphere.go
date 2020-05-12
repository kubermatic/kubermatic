/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"fmt"

	"github.com/kubermatic/kubermatic/pkg/semver"
	apimodels "github.com/kubermatic/kubermatic/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getVSphereScenarios(scenarioOptions []string, versions []*semver.Semver) []testScenario {
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

	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &vSphereScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
			customFolder:     customFolder,
			datastoreCluster: datastoreCluster,
		})
		// CoreOS
		scenarios = append(scenarios, &vSphereScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				ContainerLinux: &apimodels.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
			customFolder:     customFolder,
			datastoreCluster: datastoreCluster,
		})
		// CentOS
		scenarios = append(scenarios, &vSphereScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
			customFolder:     customFolder,
			datastoreCluster: datastoreCluster,
		})
	}

	return scenarios
}

type vSphereScenario struct {
	version          *semver.Semver
	nodeOsSpec       apimodels.OperatingSystemSpec
	customFolder     bool
	datastoreCluster bool
}

func (s *vSphereScenario) Name() string {
	return fmt.Sprintf("vsphere-%s-%s", getOSNameFromSpec(s.nodeOsSpec), s.version.String())
}

func (s *vSphereScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	spec := &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "vsphere-ger",
					Vsphere: &apimodels.VSphereCloudSpec{
						Username:  secrets.VSphere.Username,
						Password:  secrets.VSphere.Password,
						Datastore: "exsi-nas",
					},
				},
				Version: s.version.String(),
			},
		},
	}

	if s.customFolder {
		spec.Cluster.Spec.Cloud.Vsphere.Folder = "/dc-1/vm/e2e-tests/custom_folder_test"
	}

	if s.datastoreCluster {
		spec.Cluster.Spec.Cloud.Vsphere.DatastoreCluster = "dsc-1"
		spec.Cluster.Spec.Cloud.Vsphere.Datastore = ""
	}

	return spec
}

func (s *vSphereScenario) NodeDeployments(num int, _ secrets) ([]apimodels.NodeDeployment, error) {
	osName := getOSNameFromSpec(s.nodeOsSpec)
	replicas := int32(num)
	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Vsphere: &apimodels.VSphereNodeSpec{
							Template: fmt.Sprintf("machine-controller-e2e-%s", osName),
							CPUs:     2,
							Memory:   4096,
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: &s.nodeOsSpec,
				},
			},
		},
	}, nil
}

func (s *vSphereScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
