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

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/machine"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
)

type googleScenario struct {
	baseScenario
}

func (s *googleScenario) IsValid(opts *types.Options, log *zap.SugaredLogger) bool {
	if !s.baseScenario.IsValid(opts, log) {
		return false
	}

	if s.operatingSystem != providerconfig.OperatingSystemUbuntu {
		s.Log(log).Debug("Skipping because provider only supports Ubuntu.")
		return false
	}

	return true
}

func (s *googleScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
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

func (s *googleScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
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
		Version: s.version,
	}
}

func (s *googleScenario) NodeDeployments(_ context.Context, num int, secrets types.Secrets) ([]apimodels.NodeDeployment, error) {
	replicas := int32(num)

	osSpec, err := s.APIOperatingSystemSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to build OS spec: %w", err)
	}

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
					OperatingSystem: osSpec,
				},
			},
		},
	}, nil
}

func (s *googleScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	osSpec, err := s.OperatingSystemSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to build OS spec: %w", err)
	}

	nodeSpec := apiv1.NodeSpec{
		OperatingSystem: *osSpec,
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

	md, err := s.createMachineDeployment(num, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *googleScenario) getZone() string {
	return fmt.Sprintf("%s%s", s.datacenter.Spec.GCP.Region, s.datacenter.Spec.GCP.ZoneSuffixes[0])
}
