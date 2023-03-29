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
	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v3/pkg/machine/provider"

	"k8s.io/utils/pointer"
)

const (
	kubevirtImageHttpServerSvc = "http://image-repo.kube-system.svc/images"
	kubevirtCPUs               = 2
	kubevirtMemory             = "4Gi"
	kubevirtDiskSize           = "25Gi"
	kubevirtStorageClassName   = "px-csi-db"
)

type kubeVirtScenario struct {
	baseScenario
}

func (s *kubeVirtScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		ContainerRuntime: s.containerRuntime,
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Kubevirt.KKPDatacenter,
			KubeVirt: &kubermaticv1.KubeVirtCloudSpec{
				Kubeconfig: secrets.Kubevirt.Kubeconfig,
				StorageClasses: []kubermaticv1.KubeVirtInfraStorageClass{{
					Name:           kubevirtStorageClassName,
					IsDefaultClass: pointer.Bool(true),
				},
				},
			},
		},
		Version: s.clusterVersion,
	}
}

func (s *kubeVirtScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	image, err := s.getOSImage()
	if err != nil {
		return nil, err
	}

	cloudProviderSpec := provider.NewKubeVirtConfig().
		WithCPUs(kubevirtCPUs).
		WithMemory(kubevirtMemory).
		WithPrimaryDiskOSImage(image).
		WithPrimaryDiskSize(kubevirtDiskSize).
		WithPrimaryDiskStorageClassName(kubevirtStorageClassName).
		Build()

	md, err := s.createMachineDeployment(cluster, num, cloudProviderSpec, sshPubKeys)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *kubeVirtScenario) getOSImage() (string, error) {
	switch s.operatingSystem {
	case kubermaticv1.OperatingSystemUbuntu:
		return kubevirtImageHttpServerSvc + "/ubuntu-22.04.img", nil
	case kubermaticv1.OperatingSystemCentOS:
		return kubevirtImageHttpServerSvc + "/centos.img", nil
	default:
		return "", fmt.Errorf("unsupported OS %q selected", s.operatingSystem)
	}
}
