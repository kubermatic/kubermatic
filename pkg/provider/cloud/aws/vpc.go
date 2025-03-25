/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
)

func ec2VPCFilter(vpcID string) ec2types.Filter {
	return ec2types.Filter{
		Name:   ptr.To("vpc-id"),
		Values: []string{vpcID},
	}
}

func reconcileVPC(ctx context.Context, client *ec2.Client, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	vpcID := cluster.Spec.Cloud.AWS.VPCID

	// check if the VPC exists, if we have an ID cached
	if vpcID != "" {
		out, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
			VpcIds: []string{vpcID},
		})
		if err != nil && !isNotFound(err) {
			return nil, fmt.Errorf("failed to list VPCs: %w", err)
		}

		// not found
		if out == nil || len(out.Vpcs) == 0 {
			return nil, fmt.Errorf("the configured VPC %s does not exist", vpcID)
		}
	}

	// all good :)
	if vpcID != "" {
		return cluster, nil
	}

	// re-find the default VPC
	defaultVPC, err := getDefaultVPC(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("failed to get default VPC: %w", err)
	}

	return update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Cloud.AWS.VPCID = *defaultVPC.VpcId
	})
}

func getDefaultVPC(ctx context.Context, client *ec2.Client) (*ec2types.Vpc, error) {
	vpcOut, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{{
			Name:   ptr.To("isDefault"),
			Values: []string{"true"},
		}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list VPCs: %w", err)
	}

	if len(vpcOut.Vpcs) != 1 {
		return nil, errors.New("unable to find default VPC")
	}

	return &vpcOut.Vpcs[0], nil
}

func getVPCByID(ctx context.Context, client *ec2.Client, vpcID string) (*ec2types.Vpc, error) {
	vpcOut, err := client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	})

	if err != nil {
		return nil, err
	}

	if len(vpcOut.Vpcs) != 1 {
		return nil, fmt.Errorf("unable to find specified VPC with ID %q", vpcID)
	}

	return &vpcOut.Vpcs[0], nil
}
