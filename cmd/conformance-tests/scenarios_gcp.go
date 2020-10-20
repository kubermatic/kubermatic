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
	"context"
	"fmt"
	"strings"

	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/apiclient/models"
)

// Returns a matrix of (version x operating system)
func getGCPScenarios(versions []*semver.Semver) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &gcpScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
		})
		// CoreOS
		scenarios = append(scenarios, &gcpScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				ContainerLinux: &apimodels.ContainerLinuxSpec{
					// Otherwise the nodes restart directly after creation - bad for tests
					DisableAutoUpdate: true,
				},
			},
		})
		// CentOS
		scenarios = append(scenarios, &gcpScenario{
			version: v,
			nodeOsSpec: apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
		})
	}
	return scenarios
}

type gcpScenario struct {
	version    *semver.Semver
	nodeOsSpec apimodels.OperatingSystemSpec
}

func (s *gcpScenario) Name() string {
	version := strings.Replace(s.version.String(), ".", "-", -1)
	return fmt.Sprintf("gcp-%s-%s", getOSNameFromSpec(s.nodeOsSpec), version)
}

func (s *gcpScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					DatacenterName: "gcp-westeurope",
					Gcp: &apimodels.GCPCloudSpec{
						ServiceAccount: secrets.GCP.ServiceAccount,
						Network:        secrets.GCP.Network,
						Subnetwork:     secrets.GCP.Subnetwork,
					},
				},
				Version: s.version.String(),
			},
		},
	}
}

func (s *gcpScenario) NodeDeployments(ctx context.Context, num int, secrets secrets) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: &replicas,
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Gcp: &apimodels.GCPNodeSpec{
							Zone:        secrets.GCP.Zone,
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
					OperatingSystem: &s.nodeOsSpec,
				},
			},
		},
	}, nil
}

func (s *gcpScenario) OS() apimodels.OperatingSystemSpec {
	return s.nodeOsSpec
}
