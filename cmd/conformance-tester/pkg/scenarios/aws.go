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
	"errors"
	"fmt"

	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
)

const (
	awsInstanceType = "t3.small"
	awsVolumeType   = "gp2"
	awsVolumeSize   = 100
)

type awsScenario struct {
	baseScenario
}

func (s *awsScenario) compatibleOperatingSystems() sets.Set[providerconfig.OperatingSystem] {
	return sets.New[providerconfig.OperatingSystem](providerconfig.AllOperatingSystems...)
}

func (s *awsScenario) IsValid() error {
	if err := s.baseScenario.IsValid(); err != nil {
		return err
	}

	if compat := s.compatibleOperatingSystems(); !compat.Has(s.operatingSystem) {
		return fmt.Errorf("provider supports only %v", sets.List(compat))
	}

	return nil
}

func (s *awsScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.AWS.KKPDatacenter,
			AWS: &kubermaticv1.AWSCloudSpec{
				SecretAccessKey: secrets.AWS.SecretAccessKey,
				AccessKeyID:     secrets.AWS.AccessKeyID,
			},
		},
		Version: s.clusterVersion,
	}
}

func (s *awsScenario) MachineDeployments(ctx context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	vpcs, err := awsprovider.GetVPCS(ctx, secrets.AWS.AccessKeyID, secrets.AWS.SecretAccessKey, "", "", s.datacenter.Spec.AWS.Region)
	if err != nil {
		return nil, err
	}

	if len(vpcs) == 0 {
		return nil, errors.New("no VPCs found")
	}

	vpcID := vpcs[0].VpcId
	for _, vpc := range vpcs {
		if vpc.IsDefault != nil && *vpc.IsDefault {
			vpcID = vpc.VpcId
			break
		}
	}

	allSubnets, err := awsprovider.GetSubnets(ctx, secrets.AWS.AccessKeyID, secrets.AWS.SecretAccessKey, "", "", s.datacenter.Spec.AWS.Region, *vpcID)
	if err != nil {
		return nil, err
	}
	if n := len(allSubnets); n < 3 {
		return nil, fmt.Errorf("expected to get at least three subnets, got %d", n)
	}

	// Find three subnets that are in different AZs to preserve the multi az testcase
	allAZs := sets.New[string]()
	var subnets []ec2types.Subnet
	for _, subnet := range allSubnets {
		if !allAZs.Has(ptr.Deref(subnet.AvailabilityZone, "")) {
			allAZs.Insert(ptr.Deref(subnet.AvailabilityZone, ""))
			subnets = append(subnets, subnet)
		}
	}

	if n := len(subnets); n < 3 {
		return nil, fmt.Errorf("wanted three subnets in different AZs, got only %d", n)
	}

	result := []clusterv1alpha1.MachineDeployment{}

	for _, subnet := range subnets {
		cloudProviderSpec := provider.NewAWSConfig().
			WithInstanceType(awsInstanceType).
			WithDiskSize(awsVolumeSize).
			WithDiskType(awsVolumeType).
			WithAvailabilityZone(*subnet.AvailabilityZone).
			WithSubnetID(*subnet.SubnetId).
			WithSpotInstanceMaxPrice("0.5").
			Build()

		md, err := s.createMachineDeployment(cluster, num, cloudProviderSpec, sshPubKeys, secrets)
		if err != nil {
			return nil, err
		}

		result = append(result, md)
	}

	// evenly distribute the nodes among deployments
	nodesInEachAZ := num / 3
	azsWithExtraNode := num % 3

	for i := range result {
		var replicas int32
		if i < azsWithExtraNode {
			replicas = int32(nodesInEachAZ + 1)
		} else {
			replicas = int32(nodesInEachAZ)
		}
		result[i].Spec.Replicas = &replicas
	}

	return result, nil
}
