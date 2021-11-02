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

package aws

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func ec2VPCFilter(vpcID string) *ec2.Filter {
	return &ec2.Filter{
		Name:   aws.String("vpc-id"),
		Values: aws.StringSlice([]string{vpcID}),
	}
}

func reconcileVPC(client ec2iface.EC2API, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	vpcID := cluster.Spec.Cloud.AWS.VPCID

	// check if the VPC exists, if we have an ID cached
	if vpcID != "" {
		out, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{
			VpcIds: aws.StringSlice([]string{vpcID}),
		})
		if err != nil && !isNotFound(err) {
			return nil, fmt.Errorf("failed to list VPCs: %w", err)
		}

		// not found
		if out == nil || len(out.Vpcs) == 0 {
			vpcID = ""
		}
	}

	// all good :)
	if vpcID != "" {
		return cluster, nil
	}

	// re-find the default VPC
	defaultVPC, err := getDefaultVPC(client)
	if err != nil {
		return nil, fmt.Errorf("failed to get default VPC: %w", err)
	}

	return update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Cloud.AWS.VPCID = *defaultVPC.VpcId
	})
}

func getDefaultVPC(client ec2iface.EC2API) (*ec2.Vpc, error) {
	vpcOut, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{{
			Name:   aws.String("isDefault"),
			Values: aws.StringSlice([]string{"true"}),
		}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list VPCs: %w", err)
	}

	if len(vpcOut.Vpcs) != 1 {
		return nil, errors.New("unable to find default VPC")
	}

	return vpcOut.Vpcs[0], nil
}

func getVPCByID(client ec2iface.EC2API, vpcID string) (*ec2.Vpc, error) {
	vpcOut, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{ec2VPCFilter(vpcID)},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list VPCs: %w", err)
	}

	if len(vpcOut.Vpcs) != 1 {
		return nil, fmt.Errorf("unable to find specified VPC with ID %q", vpcID)
	}

	return vpcOut.Vpcs[0], nil
}
