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
	"fmt"
	"strings"

	"github.com/gophercloud/gophercloud"
	tags "github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/attributestags"
)

// TagPrefixClusterID is the prefix used for tags that track cluster ownership.
// Example: "cluster-id.k8c.io:cluster1".
const TagPrefixClusterID = "cluster-id.k8c.io:"

// TagManagedByKubermatic is the tag used to identify resources managed by Kubermatic KKP.
const TagManagedByKubermatic = "managed-by:kubermatic-kkp"

// ResourceTypeRouter is the resource type used for fetching or managing router tags in OpenStack.
const ResourceTypeRouter = "routers"

func ownersFromTags(tags []string) []string {
	clusterIDs := []string{}
	for _, tag := range tags {
		if strings.HasPrefix(tag, TagPrefixClusterID) {
			id := strings.TrimPrefix(tag, TagPrefixClusterID)
			if id != "" {
				clusterIDs = append(clusterIDs, id)
			}
		}
	}
	return clusterIDs
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
	return tags.Add(netClient, resourceType, resourceID, fmt.Sprintf("%s%s", TagPrefixClusterID, clusterID)).ExtractErr()
}

func removeOwnershipFromResource(netClient *gophercloud.ServiceClient, resourceType, resourceID, clusterID string) error {
	err := tags.Delete(netClient, resourceType, resourceID, fmt.Sprintf("%s%s", TagPrefixClusterID, clusterID)).ExtractErr()
	if err != nil && !isTagNotFound(err) {
		return err
	}
	return nil
}

func getResourceOwners(netClient *gophercloud.ServiceClient, resourceType, resourceID string) ([]string, error) {
	resourceTags, err := tags.List(netClient, resourceType, resourceID).Extract()
	if err != nil {
		return nil, err
	}
	return ownersFromTags(resourceTags), nil
}

func isTagNotFound(err error) bool {
	return strings.Contains(err.Error(), "TagNotFound")
}
