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

package provider

import (
	"context"
	"fmt"

	ec2service "github.com/aws/aws-sdk-go/service/ec2"
	ec2 "github.com/cristim/ec2-instances-info"
	"k8c.io/kubermatic/v2/pkg/resources"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"
	eksprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/eks"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Due to big amount of data we are loading AWS instance types only once. Do not edit it.
func init() {
	data, _ = ec2.Data()
}

// Region value will instruct the SDK where to make service API requests to.
// Region must be provided before a service client request is made.
const RegionEndpoint = "eu-central-1"

func listEKSClusters(cred resources.EKSCredential, region string) ([]*string, error) {
	client, err := awsprovider.GetClientSet(cred.AccessKeyID, cred.SecretAccessKey, "", "", region)
	if err != nil {
		return nil, err
	}

	return eksprovider.ListClusters(client)
}

func ListEKSClusters(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, cred resources.EKSCredential, projectID string) (apiv2.EKSClusterList, error) {
	var err error
	var clusters apiv2.EKSClusterList

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterList, err := clusterProvider.List(ctx, project)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	eksExternalCluster := make(map[string]sets.String)
	for _, externalCluster := range clusterList.Items {
		cloud := externalCluster.Spec.CloudSpec
		if cloud != nil && cloud.EKS != nil {
			region := cloud.EKS.Region
			if _, ok := eksExternalCluster[region]; !ok {
				eksExternalCluster[region] = make(sets.String)
			}
			eksExternalCluster[region] = eksExternalCluster[region].Insert(cloud.EKS.Name)
		}
	}

	region := cred.Region
	// list EKS clusters for user specified region
	if region != "" {
		eksClusters, err := listEKSClusters(cred, region)
		if err != nil {
			return nil, fmt.Errorf("cannot list clusters: %w", err)
		}

		for _, f := range eksClusters {
			var imported bool
			if clusterSet, ok := eksExternalCluster[region]; ok {
				if clusterSet.Has(*f) {
					imported = true
				}
			}
			clusters = append(clusters, apiv2.EKSCluster{Name: *f, Region: region, IsImported: imported})
		}
	}

	return clusters, nil
}

func ListEKSSubnetIDs(ctx context.Context, cred resources.EKSCredential, vpcID string) (apiv2.EKSSubnetIDList, error) {
	subnetIDs := apiv2.EKSSubnetIDList{}

	subnetResults, err := awsprovider.GetSubnets(ctx, cred.AccessKeyID, cred.SecretAccessKey, "", "", cred.Region, vpcID)
	if err != nil {
		return nil, err
	}

	for _, subnet := range subnetResults {
		subnetIDs = append(subnetIDs, apiv2.EKSSubnetID(*subnet.SubnetId))
	}
	return subnetIDs, nil
}

func ListEKSVPC(ctx context.Context, cred resources.EKSCredential) (apiv2.EKSVPCList, error) {
	vpcs := apiv2.EKSVPCList{}

	vpcResults, err := awsprovider.GetVPCS(ctx, cred.AccessKeyID, cred.SecretAccessKey, "", "", cred.Region)
	if err != nil {
		return nil, err
	}

	for _, v := range vpcResults {
		vpc := apiv2.EKSVPC{
			ID:        *v.VpcId,
			IsDefault: *v.IsDefault,
		}
		vpcs = append(vpcs, vpc)
	}
	return vpcs, nil
}

func ListInstanceTypes(ctx context.Context, cred resources.EKSCredential) (apiv2.EKSInstanceTypeList, error) {
	instanceTypes := apiv2.EKSInstanceTypeList{}

	if data == nil {
		return nil, fmt.Errorf("AWS instance type data not initialized")
	}

	instanceTypesResults, err := awsprovider.GetInstanceTypes(cred.AccessKeyID, cred.SecretAccessKey, "", "", cred.Region)
	if err != nil {
		return nil, err
	}

	for _, i := range *data {
		for _, r := range instanceTypesResults {
			if i.InstanceType == *r.InstanceType {
				instanceTypes = append(instanceTypes, apiv2.EKSInstanceType{
					Name:       i.InstanceType,
					PrettyName: i.PrettyName,
					Memory:     i.Memory,
					VCPUs:      i.VCPU,
					GPUs:       i.GPU,
				})
				break
			}
		}
	}

	return instanceTypes, nil
}

func ListEKSRegions(ctx context.Context, cred resources.EKSCredential) (apiv2.EKSRegionList, error) {
	regionInput := &ec2service.DescribeRegionsInput{}

	// Must provide either a region or endpoint configured to use the SDK, even for operations that may enumerate other regions
	// See https://github.com/aws/aws-sdk-go/issues/224 for more details
	client, err := awsprovider.GetClientSet(cred.AccessKeyID, cred.SecretAccessKey, "", "", RegionEndpoint)
	if err != nil {
		return nil, err
	}

	// Retrieves all regions/endpoints that work with EC2
	regionOutput, err := client.EC2.DescribeRegions(regionInput)
	if err != nil {
		return nil, fmt.Errorf("cannot list regions: %w", err)
	}

	var regionList []string
	for _, region := range regionOutput.Regions {
		regionList = append(regionList, *region.RegionName)
	}
	return regionList, nil
}

func ListEKSSecurityGroupIDs(ctx context.Context, cred resources.EKSCredential, vpcID string) (apiv2.EKSSecurityGroupIDList, error) {
	securityGroupID := apiv2.EKSSecurityGroupIDList{}

	securityGroups, err := awsprovider.GetSecurityGroupsByVPC(ctx, cred.AccessKeyID, cred.SecretAccessKey, "", "", cred.Region, vpcID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get security groups: %w", err)
	}

	for _, group := range securityGroups {
		securityGroupID = append(securityGroupID, apiv2.EKSSecurityGroupID(*group.GroupId))
	}
	return securityGroupID, nil
}
