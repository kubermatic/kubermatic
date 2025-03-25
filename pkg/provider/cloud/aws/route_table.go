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
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/utils/ptr"
)

func reconcileRouteTable(ctx context.Context, client *ec2.Client, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	vpcID := cluster.Spec.Cloud.AWS.VPCID
	tableID := cluster.Spec.Cloud.AWS.RouteTableID

	// check if the RT exists, if we have an ID cached
	if tableID != "" {
		out, err := client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
			RouteTableIds: []string{cluster.Spec.Cloud.AWS.RouteTableID},
			Filters:       []ec2types.Filter{ec2VPCFilter(vpcID)},
		})
		if err != nil && !isNotFound(err) {
			return nil, fmt.Errorf("failed to list route tables: %w", err)
		}

		// not found
		if out == nil || len(out.RouteTables) == 0 {
			tableID = ""
		}
	}

	// all good :)
	if tableID != "" {
		return cluster, nil
	}

	// re-find the default route table
	table, err := getDefaultRouteTable(ctx, client, vpcID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default route table: %w", err)
	}

	return update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Cloud.AWS.RouteTableID = *table.RouteTableId
	})
}

func getDefaultRouteTable(ctx context.Context, client *ec2.Client, vpcID string) (*ec2types.RouteTable, error) {
	out, err := client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []ec2types.Filter{
			ec2VPCFilter(vpcID),
			{
				Name:   ptr.To("association.main"),
				Values: []string{"true"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list route tables: %w", err)
	}

	if len(out.RouteTables) != 1 {
		return nil, fmt.Errorf("could not get default route table in VPC %s; make sure you have exactly one main route table for the VPC", vpcID)
	}

	return &out.RouteTables[0], nil
}
