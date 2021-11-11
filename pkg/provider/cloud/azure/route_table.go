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

package azure

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
)

func routeTableName(cluster *kubermaticv1.Cluster) string {
	return resourceNamePrefix + cluster.Name
}

// ensureRouteTable will create or update an Azure route table attached to the specified subnet. The call is idempotent.
func ensureRouteTable(ctx context.Context, cloud kubermaticv1.CloudSpec, location string, credentials Credentials) error {
	routeTablesClient, err := getRouteTablesClient(cloud, credentials)
	if err != nil {
		return err
	}

	parameters := network.RouteTable{
		Name:     to.StringPtr(cloud.Azure.RouteTableName),
		Location: to.StringPtr(location),
		RouteTablePropertiesFormat: &network.RouteTablePropertiesFormat{
			Subnets: &[]network.Subnet{
				{
					Name: to.StringPtr(cloud.Azure.SubnetName),
					ID:   to.StringPtr(assembleSubnetID(cloud)),
				},
			},
		},
	}

	future, err := routeTablesClient.CreateOrUpdate(ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName, parameters)
	if err != nil {
		return fmt.Errorf("failed to create or update route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	if err = future.WaitForCompletionRef(ctx, routeTablesClient.Client); err != nil {
		return fmt.Errorf("failed to create or update route table %q: %v", cloud.Azure.RouteTableName, err)
	}

	return nil
}

func deleteRouteTable(ctx context.Context, cloud kubermaticv1.CloudSpec, credentials Credentials) error {
	routeTablesClient, err := getRouteTablesClient(cloud, credentials)
	if err != nil {
		return err
	}

	future, err := routeTablesClient.Delete(ctx, cloud.Azure.ResourceGroup, cloud.Azure.RouteTableName)
	if err != nil {
		return err
	}

	if err = future.WaitForCompletionRef(ctx, routeTablesClient.Client); err != nil {
		return err
	}

	return nil
}
