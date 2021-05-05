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

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	utilpointer "k8s.io/utils/pointer"
)

// Returns a matrix of (version x operating system)
func getKubevirtScenarios(versions []*semver.Semver, log *zap.SugaredLogger) []testScenario {
	var scenarios []testScenario
	for _, v := range versions {
		// Ubuntu
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: &apimodels.OperatingSystemSpec{
				Ubuntu: &apimodels.UbuntuSpec{},
			},
			logger: log,
		})
		// CentOS
		scenarios = append(scenarios, &kubevirtScenario{
			version: v,
			nodeOsSpec: &apimodels.OperatingSystemSpec{
				Centos: &apimodels.CentOSSpec{},
			},
			logger: log,
		})
	}

	return scenarios
}

type kubevirtScenario struct {
	version    *semver.Semver
	nodeOsSpec *apimodels.OperatingSystemSpec
	logger     *zap.SugaredLogger
}

func (s *kubevirtScenario) Name() string {
	return fmt.Sprintf("kubevirt-%s-%s", getOSNameFromSpec(*s.nodeOsSpec), s.version.String())
}

func (s *kubevirtScenario) Cluster(secrets secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Type: "kubernetes",
			Spec: &apimodels.ClusterSpec{
				Cloud: &apimodels.CloudSpec{
					Kubevirt: &apimodels.KubevirtCloudSpec{
						Kubeconfig: secrets.Kubevirt.Kubeconfig,
					},
					DatacenterName: "kubevirt-europe-west3-c",
				},
				Version: s.version.String(),
			},
		},
	}
}

func (s *kubevirtScenario) NodeDeployments(_ context.Context, num int, _ secrets) ([]apimodels.NodeDeployment, error) {
	var sourceURL string

	switch {
	case s.nodeOsSpec.Ubuntu != nil:
		sourceURL = "http://10.102.236.197/ubuntu.img"
	case s.nodeOsSpec.Centos != nil:
		sourceURL = "http://10.102.236.197/centos.img"
	default:
		s.logger.Error("coreos operating system is not supported")
	}

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: utilpointer.Int32Ptr(int32(num)),
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Kubevirt: &apimodels.KubevirtNodeSpec{
							Memory:           utilpointer.StringPtr("2048M"),
							Namespace:        utilpointer.StringPtr("kube-system"),
							SourceURL:        utilpointer.StringPtr(sourceURL),
							StorageClassName: utilpointer.StringPtr("kubermatic-fast"),
							PVCSize:          utilpointer.StringPtr("25Gi"),
							CPUs:             utilpointer.StringPtr("2"),
						},
					},
					Versions: &apimodels.NodeVersionInfo{
						Kubelet: s.version.String(),
					},
					OperatingSystem: s.nodeOsSpec,
				},
			},
		},
	}, nil
}

func (s *kubevirtScenario) OS() apimodels.OperatingSystemSpec {
	return *s.nodeOsSpec
}
