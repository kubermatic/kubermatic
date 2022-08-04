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
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/machine"
	"k8c.io/kubermatic/v2/pkg/semver"
	apimodels "k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"

	utilpointer "k8s.io/utils/pointer"
)

const (
	kubevirtImageHttpServerSvc = "http://image-repo.kube-system.svc.cluster.local/images"
	kubevirtCPUs               = "2"
	kubevirtMemory             = "4Gi"
	kubevirtDiskSize           = "25Gi"
	kubevirtDiskClassName      = "longhorn"
)

// GetKubevirtScenarios Returns a matrix of (version x operating system).
func GetKubevirtScenarios(versions []*semver.Semver, log *zap.SugaredLogger, datacenter *kubermaticv1.Datacenter) []Scenario {
	baseScenarios := []*kubevirtScenario{
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

type kubevirtScenario struct {
	baseScenario

	logger *zap.SugaredLogger
}

func (s *kubevirtScenario) DeepCopy() *kubevirtScenario {
	return &kubevirtScenario{
		baseScenario: *s.baseScenario.DeepCopy(),
		logger:       s.logger,
	}
}

func (s *kubevirtScenario) Name() string {
	return fmt.Sprintf("kubevirt-%s-%s-%s", getOSNameFromSpec(s.osSpec), s.containerRuntime, s.version.String())
}

func (s *kubevirtScenario) APICluster(secrets types.Secrets) *apimodels.CreateClusterSpec {
	return &apimodels.CreateClusterSpec{
		Cluster: &apimodels.Cluster{
			Spec: &apimodels.ClusterSpec{
				ContainerRuntime: s.containerRuntime,
				Cloud: &apimodels.CloudSpec{
					DatacenterName: secrets.Kubevirt.KKPDatacenter,
					Kubevirt: &apimodels.KubevirtCloudSpec{
						Kubeconfig: secrets.Kubevirt.Kubeconfig,
					},
				},
				Version: apimodels.Semver(s.version.String()),
			},
		},
	}
}

func (s *kubevirtScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Kubevirt.KKPDatacenter,
			Kubevirt: &kubermaticv1.KubevirtCloudSpec{
				Kubeconfig: secrets.Kubevirt.Kubeconfig,
			},
		},
		Version: *s.version,
	}
}

func (s *kubevirtScenario) NodeDeployments(_ context.Context, num int, _ types.Secrets) ([]apimodels.NodeDeployment, error) {
	image, err := s.getOSImage()
	if err != nil {
		return nil, err
	}

	return []apimodels.NodeDeployment{
		{
			Spec: &apimodels.NodeDeploymentSpec{
				Replicas: utilpointer.Int32Ptr(int32(num)),
				Template: &apimodels.NodeSpec{
					Cloud: &apimodels.NodeCloudSpec{
						Kubevirt: &apimodels.KubevirtNodeSpec{
							CPUs:                        utilpointer.StringPtr(kubevirtCPUs),
							Memory:                      utilpointer.StringPtr(kubevirtMemory),
							PrimaryDiskOSImage:          utilpointer.String(image),
							PrimaryDiskSize:             utilpointer.String(kubevirtDiskSize),
							PrimaryDiskStorageClassName: utilpointer.String(kubevirtDiskClassName),
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

func (s *kubevirtScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	image, err := s.getOSImage()
	if err != nil {
		return nil, err
	}

	nodeSpec := apiv1.NodeSpec{
		Cloud: apiv1.NodeCloudSpec{
			Kubevirt: &apiv1.KubevirtNodeSpec{
				CPUs:                        kubevirtCPUs,
				Memory:                      kubevirtMemory,
				PrimaryDiskOSImage:          image,
				PrimaryDiskSize:             kubevirtDiskSize,
				PrimaryDiskStorageClassName: kubevirtDiskClassName,
			},
		},
	}

	config, err := machine.GetKubevirtProviderConfig(cluster, nodeSpec, s.datacenter)
	if err != nil {
		return nil, err
	}

	md, err := createMachineDeployment(num, s.version, getOSNameFromSpec(s.osSpec), s.osSpec, providerconfig.CloudProviderKubeVirt, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *kubevirtScenario) getOSImage() (string, error) {
	os := getOSNameFromSpec(s.osSpec)

	switch {
	case os == providerconfig.OperatingSystemUbuntu:
		return kubevirtImageHttpServerSvc + "/ubuntu.img", nil
	case os == providerconfig.OperatingSystemCentOS:
		return kubevirtImageHttpServerSvc + "/centos.img", nil
	default:
		return "", fmt.Errorf("unsupported OS %q selected", os)
	}
}
