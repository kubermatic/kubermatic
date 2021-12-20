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

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	awsprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/aws"

	"k8s.io/apimachinery/pkg/util/sets"
)

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
