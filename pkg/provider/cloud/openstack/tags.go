/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package openstack

import (
	"strings"

	"github.com/gophercloud/gophercloud"
	tags "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"

	"k8s.io/apimachinery/pkg/util/sets"
)

// TagPrefixOwnedByCluster is the prefix used for tags that track cluster ownership.
// Example: "owned-by-cluster:cluster1,cluster2".
const TagPrefixOwnedByCluster = "owned-by-cluster:"

// TagManagedByKubermatic is the tag used to identify resources managed by Kubermatic KKP.
const TagManagedByKubermatic = "managed-by:kubermatic-kkp"

// ResourceTypeRouter is the resource type used for fetching or managing router tags in OpenStack.
const ResourceTypeRouter = "routers"

// addTagOwnership adds a cluster name to the "owned-by-cluster" tag in the tags slice.
// If the "owned-by-cluster" tag exists, it appends the clusterName to the existing IDs (if not already present).
// If the "owned-by-cluster" tag does not exist, it returns the original tags slice unchanged.
func addTagOwnership(tags []string, clusterName string) []string {
	for i, tag := range tags {
		if strings.HasPrefix(tag, TagPrefixOwnedByCluster) {
			existingIDs := strings.TrimPrefix(tag, TagPrefixOwnedByCluster)
			idSet := sets.New[string]()
			for _, id := range strings.Split(existingIDs, ",") {
				if id != "" {
					idSet.Insert(id)
				}
			}
			idSet.Insert(clusterName)
			tags[i] = TagPrefixOwnedByCluster + strings.Join(sets.List(idSet), ",")
			return tags
		}
	}
	return tags
}

// removeTagOwnership removes a cluster name from the "owned-by-cluster" tag in the tags slice.
// If the "owned-by-cluster" tag exists, it removes the clusterName from the existing IDs.
// If the "owned-by-cluster" tag does not exist or the clusterName is not present, it returns the original tags slice unchanged.
func removeTagOwnership(tags []string, clusterName string) []string {
	for i, tag := range tags {
		if strings.HasPrefix(tag, TagPrefixOwnedByCluster) {
			existingIDs := strings.TrimPrefix(tag, TagPrefixOwnedByCluster)
			idSet := sets.New[string]()
			for _, id := range strings.Split(existingIDs, ",") {
				if id != "" {
					idSet.Insert(id)
				}
			}
			// Remove the clusterName from the set
			idSet.Delete(clusterName)

			// If no IDs remain, remove the "owned-by-cluster" tag entirely
			if idSet.Len() == 0 {
				return append(tags[:i], tags[i+1:]...)
			}

			tags[i] = TagPrefixOwnedByCluster + strings.Join(sets.List(idSet), ",")
			return tags
		}
	}
	return tags
}

func ownersFromTags(tags []string) []string {
	for _, tag := range tags {
		if strings.HasPrefix(tag, TagPrefixOwnedByCluster) {
			parts := strings.TrimPrefix(tag, TagPrefixOwnedByCluster)
			ClusterIDs := []string{}
			for _, id := range strings.Split(parts, ",") {
				if id != "" {
					ClusterIDs = append(ClusterIDs, id)
				}
			}
			return ClusterIDs
		}
	}
	return []string{}
}

// isManagedResource checks if a resource is managed by Kubermatic KKP.
// Returns true only if:
// - The resource exists, and It has the tag "managed-by:kubermatic-kkp"
// Returns false if:
// - The resource doesn't exist or The tag check fails due to API errors, or The tag is explicitly not present.
func isManagedResource(netClient *gophercloud.ServiceClient, resourceType, resourceID string) bool {
	res, _ := tags.Confirm(netClient, resourceType, resourceID, TagManagedByKubermatic).Extract()
	return res
}

func addOwnershipToResource(netClient *gophercloud.ServiceClient, resourceType, resourceID, clusterID string) error {
	resourceTags, err := tags.List(netClient, resourceType, resourceID).Extract()
	if err != nil {
		return err
	}
	newTags := addTagOwnership(resourceTags, clusterID)
	return tags.ReplaceAll(netClient, resourceType, resourceID, tags.ReplaceAllOpts{Tags: newTags}).Err
}

func removeOwnershipFromResource(netClient *gophercloud.ServiceClient, resourceType, resourceID, clusterID string) error {
	resourceTags, err := tags.List(netClient, resourceType, resourceID).Extract()
	if err != nil {
		return err
	}
	newTags := removeTagOwnership(resourceTags, clusterID)
	return tags.ReplaceAll(netClient, resourceType, resourceID, tags.ReplaceAllOpts{Tags: newTags}).Err
}

func getResourceOwners(netClient *gophercloud.ServiceClient, resourceType, resourceID string) ([]string, error) {
	resourceTags, err := tags.List(netClient, resourceType, resourceID).Extract()
	if err != nil {
		return nil, err
	}
	return ownersFromTags(resourceTags), nil
}
