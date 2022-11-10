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

package nutanix

import (
	"context"
	"fmt"
	"strings"

	nutanixv3 "github.com/nutanix-cloud-native/prism-go-client/v3"

	"k8s.io/utils/pointer"
)

func GetClusters(ctx context.Context, client *ClientSet) ([]nutanixv3.ClusterIntentResponse, error) {
	resp, err := client.Prism.V3.ListAllCluster(ctx, "")
	if err != nil {
		return nil, wrapNutanixError(err)
	}

	var clusters []nutanixv3.ClusterIntentResponse

	if resp != nil {
		for _, entity := range resp.Entities {
			if entity != nil {
				clusters = append(clusters, *entity)
			}
		}
	}

	return clusters, nil
}

func GetProjects(ctx context.Context, client *ClientSet) ([]nutanixv3.Project, error) {
	resp, err := client.Prism.V3.ListAllProject(ctx, "")
	if err != nil {
		return nil, wrapNutanixError(err)
	}

	var projects []nutanixv3.Project

	if resp != nil {
		for _, entity := range resp.Entities {
			if entity != nil {
				projects = append(projects, *entity)
			}
		}
	}

	return projects, nil
}

func GetSubnets(ctx context.Context, client *ClientSet, clusterName, projectName string) ([]nutanixv3.SubnetIntentResponse, error) {
	resp, err := client.Prism.V3.ListAllSubnet(ctx, "", nil)
	if err != nil {
		return nil, wrapNutanixError(err)
	}

	var (
		subnets             []nutanixv3.SubnetIntentResponse
		projectAllowedUUIDs []string
	)

	// get all subnets associated with the project. We _could_ return that as a list, but we
	// want to return the full information of subnets, so we only use this list of names as
	// constraints for our loop later on.
	if projectName != "" {
		project, err := GetProjectByName(ctx, client, projectName)
		if err != nil {
			return nil, err
		}

		for _, subnetRef := range project.Spec.Resources.SubnetReferenceList {
			if subnetRef != nil {
				projectAllowedUUIDs = append(projectAllowedUUIDs, subnetRef.UUID)
			}
		}
	}

	cluster, err := GetClusterByName(ctx, client, clusterName)
	if err != nil {
		return nil, err
	}

	if cluster.Metadata == nil || cluster.Metadata.UUID == nil {
		return nil, fmt.Errorf("Cluster '%s' has no valid metadata", clusterName)
	}

	if resp != nil {
		for _, entity := range resp.Entities {
			if entity != nil {
				if entity.Status != nil &&
					(entity.Status.ClusterReference == nil || (entity.Status.ClusterReference.UUID != nil && *entity.Status.ClusterReference.UUID == *cluster.Metadata.UUID)) &&
					(projectName == "" || projectName == DefaultProject || contains(projectAllowedUUIDs, *entity.Metadata.UUID)) {
					subnets = append(subnets, *entity)
				}
			}
		}
	}

	return subnets, nil
}

func GetCategories(ctx context.Context, client *ClientSet) ([]nutanixv3.CategoryKeyStatus, error) {
	resp, err := client.Prism.V3.ListCategories(ctx, &nutanixv3.CategoryListMetadata{Kind: pointer.String("category")})
	if err != nil {
		return nil, wrapNutanixError(err)
	}

	var categoryKeys []nutanixv3.CategoryKeyStatus

	if resp != nil {
		for _, entity := range resp.Entities {
			if entity != nil {
				categoryKeys = append(categoryKeys, *entity)
			}
		}
	}

	return categoryKeys, nil
}

func GetCategoryValues(ctx context.Context, client *ClientSet, category string) ([]nutanixv3.CategoryValueStatus, error) {
	resp, err := client.Prism.V3.ListAllCategoryValues(ctx, category, "")
	if err != nil {
		return nil, wrapNutanixError(err)
	}

	var categoryValues []nutanixv3.CategoryValueStatus

	if resp != nil {
		for _, entity := range resp.Entities {
			if entity != nil {
				categoryValues = append(categoryValues, *entity)
			}
		}
	}

	return categoryValues, nil
}

func wrapNutanixError(initialErr error) error {
	if initialErr == nil {
		return nil
	}

	resp, err := ParseNutanixError(initialErr)
	if err != nil {
		// failed to parse error, let's make sure it doesn't have new lines at least.
		return fmt.Errorf("api error: %s", strings.ReplaceAll(initialErr.Error(), "\n", ""))
	}

	var msgs []string
	for _, msg := range resp.MessageList {
		msgs = append(msgs, fmt.Sprintf("%s: %s", msg.Message, msg.Reason))
	}

	return fmt.Errorf("api error (%s, code %d): %s", resp.State, resp.Code, strings.Join(msgs, ", "))
}

func contains(sArr []string, s string) bool {
	for _, elem := range sArr {
		if elem == s {
			return true
		}
	}

	return false
}
