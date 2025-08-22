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
)

type awsScenario struct {
	BaseScenario
}

func (s *awsScenario) compatibleOperatingSystems() sets.Set[providerconfig.OperatingSystem] {
	return sets.New[providerconfig.OperatingSystem](
		providerconfig.OperatingSystemUbuntu,
		providerconfig.OperatingSystemRHEL,
		providerconfig.OperatingSystemFlatcar,
		providerconfig.OperatingSystemRockyLinux,
	)
}

func (s *awsScenario) IsValid() error {
	if err := s.IsValid(); err != nil {
		return err
	}

	if compat := s.compatibleOperatingSystems(); !compat.Has(s.OperatingSystem()) {
		return fmt.Errorf("provider supports only %v", sets.List(compat))
	}

	return nil
}

func (s *awsScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.AWS.KKPDatacenter,
			AWS: &kubermaticv1.AWSCloudSpec{
				AccessKeyID:     secrets.AWS.AccessKeyID,
				SecretAccessKey: secrets.AWS.SecretAccessKey,
			},
		},
		Version: s.ClusterVersion(),
	}
}

func (s *awsScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	ami, err := s.getAMI(s.Datacenter().Spec.AWS.Region)
	if err != nil {
		return nil, fmt.Errorf("failed to get AMI: %w", err)
	}

	cloudProviderSpec := provider.NewAWSConfig().
		WithAMI(ami).
		WithInstanceType("t2.medium").
		WithDiskSize(50).
		WithAvailabilityZone(s.Datacenter().Spec.AWS.Region + "a").
		Build()

	md, err := s.CreateMachineDeployment(cluster, num, cloudProviderSpec, sshPubKeys, secrets)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

func (s *awsScenario) getAMI(region string) (string, error) {
	switch s.OperatingSystem() {
	case providerconfig.OperatingSystemUbuntu:
		return getUbuntuAMI(region)
	case providerconfig.OperatingSystemFlatcar:
		return getFlatcarAMI(region)
	default:
		return "", fmt.Errorf("unsupported OS %q selected", s.OperatingSystem())
	}
}

// Those are the AMIs from https://github.com/kubermatic/machine-controller/blob/main/examples/machinedeployment-aws.yaml
// For other Ubuntu AMIs please see the official finder at https://cloud-images.ubuntu.com/locator/ec2/
func getUbuntuAMI(region string) (string, error) {
	switch region {
	case "eu-central-1":
		return "ami-0e731c03a84422e33", nil
	case "us-east-1":
		return "ami-0ea6231df37626256", nil
	default:
		return "", fmt.Errorf("no Ubuntu AMI for region %q configured", region)
	}
}

// For other Flatcar AMIs please see the official finder at https://www.flatcar.org/docs/latest/installing/cloud/aws-ec2/
func getFlatcarAMI(region string) (string, error) {
	switch region {
	case "eu-central-1":
		return "ami-06a3b03e387903967", nil
	case "us-east-1":
		return "ami-02f33d5b33c4c1336", nil
	default:
		return "", fmt.Errorf("no Flatcar AMI for region %q configured", region)
	}
}
