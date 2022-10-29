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
	"k8c.io/kubermatic/v2/pkg/resources/machine"
)

const (
	kubevirtImageHttpServerSvc = "http://image-repo.kube-system.svc.cluster.local/images"
	kubevirtCPUs               = "2"
	kubevirtMemory             = "4Gi"
	kubevirtDiskSize           = "25Gi"
	kubevirtDiskClassName      = "longhorn"
)

type kubevirtScenario struct {
	baseScenario
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
		Version: s.version,
	}
}

func (s *kubevirtScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster) ([]clusterv1alpha1.MachineDeployment, error) {
	image, err := s.getOSImage()
	if err != nil {
		return nil, err
	}

	osSpec, err := s.OperatingSystemSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to build OS spec: %w", err)
	}

	nodeSpec := apiv1.NodeSpec{
		OperatingSystem: *osSpec,
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

	md, err := s.createMachineDeployment(num, config)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *kubevirtScenario) getOSImage() (string, error) {
	switch s.operatingSystem {
	case providerconfig.OperatingSystemUbuntu:
		return kubevirtImageHttpServerSvc + "/ubuntu-22.04.img", nil
	case providerconfig.OperatingSystemCentOS:
		return kubevirtImageHttpServerSvc + "/centos.img", nil
	default:
		return "", fmt.Errorf("unsupported OS %q selected", s.operatingSystem)
	}
}
