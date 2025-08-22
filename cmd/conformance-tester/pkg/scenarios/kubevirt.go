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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

const (
	kubevirtImageHTTPServerSvc = "https://cloud-images.ubuntu.com/noble/current"
	kubevirtVCPUs              = 2
	kubevirtMemory             = "4Gi"
	kubevirtDiskSize           = "25Gi"
	kubevirtStorageClassName   = "longhorn"
)

type KubevirtScenario struct {
	BaseScenario
	cpu      string
	memory   string
	pvcSize  string
	pvcSC    string
	template string
}

func (s *KubevirtScenario) compatibleOperatingSystems() sets.Set[providerconfig.OperatingSystem] {
	return sets.New(
		providerconfig.OperatingSystemUbuntu,
		providerconfig.OperatingSystemRHEL,
		providerconfig.OperatingSystemFlatcar,
		providerconfig.OperatingSystemRockyLinux,
	)
}

func (s *KubevirtScenario) IsValid() error {
	if err := s.IsValid(); err != nil {
		return err
	}

	if compat := s.compatibleOperatingSystems(); !compat.Has(s.operatingSystem) {
		return fmt.Errorf("provider supports only %v", sets.List(compat))
	}

	return nil
}

func (s *KubevirtScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Kubevirt.KKPDatacenter,
			Kubevirt: &kubermaticv1.KubevirtCloudSpec{
				Kubeconfig: secrets.Kubevirt.Kubeconfig,
				StorageClasses: []kubermaticv1.KubeVirtInfraStorageClass{{
					Name:           kubevirtStorageClassName,
					IsDefaultClass: ptr.To(true),
				}},
			},
		},
		Version: s.clusterVersion,
	}
}

func (s *KubevirtScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	image, err := s.getOSImage()
	if err != nil {
		return nil, err
	}

	cloudProviderSpec := provider.NewKubevirtConfig().
		WithVCPUs(kubevirtVCPUs).
		WithMemory(kubevirtMemory).
		WithPrimaryDiskOSImage(image).
		WithPrimaryDiskSize(kubevirtDiskSize).
		WithPrimaryDiskStorageClassName(kubevirtStorageClassName).
		Build()

	md, err := s.CreateMachineDeployment(cluster, num, cloudProviderSpec, sshPubKeys, secrets)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *KubevirtScenario) getOSImage() (string, error) {
	switch s.operatingSystem {
	case providerconfig.OperatingSystemUbuntu:
		return kubevirtImageHTTPServerSvc + "/noble-server-cloudimg-amd64.img", nil
	default:
		return "", fmt.Errorf("unsupported OS %q selected", s.operatingSystem)
	}
}
