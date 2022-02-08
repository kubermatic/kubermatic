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

	"github.com/aws/aws-sdk-go/service/eks"

	ec2service "github.com/aws/aws-sdk-go/service/ec2"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"

	"k8s.io/apimachinery/pkg/util/sets"
)

// Region value will instruct the SDK where to make service API requests to.
// Region must be provided before a service client request is made.
const RegionEndpoint = "eu-central-1"

type EKSCredential struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
}

func listEKSClusters(cred EKSCredential, region string) ([]*string, error) {
	client, err := awsprovider.GetClientSet(cred.AccessKeyID, cred.SecretAccessKey, "", "", region)
	if err != nil {
		return nil, err
	}

	req, res := client.EKS.ListClustersRequest(&eks.ListClustersInput{})
	err = req.Send()
	if err != nil {
		return nil, fmt.Errorf("cannot list clusters in region=%s: %w", region, err)
	}

	return res.Clusters, nil
}

func ListEKSClusters(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, cred EKSCredential, projectID string) (apiv2.EKSClusterList, error) {
	var err error
	var clusters apiv2.EKSClusterList

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterList, err := clusterProvider.List(project)
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

func ValidateEKSCredentials(ctx context.Context, credential EKSCredential) error {
	client, err := awsprovider.GetClientSet(credential.AccessKeyID, credential.SecretAccessKey, "", "", credential.Region)
	if err != nil {
		return err
	}

	_, err = client.EKS.ListClusters(&eks.ListClustersInput{})

	return err
}

func ListEKSSubnetIDs(ctx context.Context, cred EKSCredential, vpcID string) (apiv2.EKSSubnetIDList, error) {
	subnetIDs := apiv2.EKSSubnetIDList{}

	subnetResults, err := awsprovider.GetSubnets(cred.AccessKeyID, cred.SecretAccessKey, "", "", cred.Region, vpcID)
	if err != nil {
		return nil, fmt.Errorf("couldn't get subnets: %w", err)
	}

	for _, subnet := range subnetResults {
		subnetIDs = append(subnetIDs, apiv2.EKSSubnetID(*subnet.SubnetId))
	}
	return subnetIDs, nil
}

func ListEKSVpcIds(ctx context.Context, cred EKSCredential) (apiv2.EKSVpcIdList, error) {
	vpcIDs := apiv2.EKSVpcIdList{}

	vpcResults, err := awsprovider.GetVPCS(cred.AccessKeyID, cred.SecretAccessKey, "", "", cred.Region)
	if err != nil {
		return nil, fmt.Errorf("couldn't get vpcs: %w", err)
	}

	for _, v := range vpcResults {
		vpcIDs = append(vpcIDs, apiv2.EKSVpcId(*v.VpcId))
	}
	return vpcIDs, nil
}

func ListEKSRegions(ctx context.Context, cred EKSCredential) (apiv2.EKSRegions, error) {
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
