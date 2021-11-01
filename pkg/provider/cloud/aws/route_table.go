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
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2/ec2iface"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func reconcileRouteTable(client ec2iface.EC2API, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	vpcID := cluster.Spec.Cloud.AWS.VPCID
	tableID := cluster.Spec.Cloud.AWS.RouteTableID

	// check if the RT exists, if we have an ID cached
	if tableID != "" {
		out, err := client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
			RouteTableIds: aws.StringSlice([]string{cluster.Spec.Cloud.AWS.RouteTableID}),
			Filters:       []*ec2.Filter{ec2VPCFilter(vpcID)},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list route tables: %w", err)
		}

		// not found
		if len(out.RouteTables) == 0 {
			tableID = ""
		}
	}

	// all good :)
	if tableID != "" {
		return cluster, nil
	}

	// re-find the default route table
	table, err := getDefaultRouteTable(client, vpcID)
	if err != nil {
		return nil, fmt.Errorf("failed to get default route table: %w", err)
	}

	return update(cluster.Name, func(cluster *kubermaticv1.Cluster) {
		cluster.Spec.Cloud.AWS.RouteTableID = *table.RouteTableId
	})
}

func getDefaultRouteTable(client ec2iface.EC2API, vpcID string) (*ec2.RouteTable, error) {
	out, err := client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			ec2VPCFilter(vpcID),
			{
				Name:   aws.String("association.main"),
				Values: aws.StringSlice([]string{"true"}),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list route tables: %w", err)
	}

	if len(out.RouteTables) != 1 {
		return nil, fmt.Errorf("could not get default route table in VPC %s; make sure you have exactly one main route table for the VPC", vpcID)
	}

	return out.RouteTables[0], nil
}
