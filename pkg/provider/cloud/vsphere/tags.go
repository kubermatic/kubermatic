/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package vsphere

import (
	"context"
	"fmt"
	"strings"

	vapitags "github.com/vmware/govmomi/vapi/tags"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	// managedTagsAnnotation is an annotation on the KKP Cluster object that stores a comma-separated list of vSphere tag IDs
	// managed by this cluster.
	managedTagsAnnotation = "kubermatic.io/vsphere-managed-tags"
)

func reconcileTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	// The update function can be called multiple times, so we need to ensure we have the latest version of the cluster object.
	var err error
	managedTags, err := syncCreatedClusterTags(ctx, restSession, cluster)
	if err != nil {
		return nil, fmt.Errorf("failed to sync created tags: %w", err)
	}

	managedTags, err = syncDeletedClusterTags(ctx, restSession, cluster, managedTags)
	if err != nil {
		return nil, fmt.Errorf("failed to sync deleted tags: %w", err)
	}

	updatedTags := strings.Join(sets.List(managedTags), ",")

	return update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		if !kuberneteshelper.HasFinalizer(cluster, tagCleanupFinalizer) {
			kuberneteshelper.AddFinalizer(cluster, tagCleanupFinalizer)
		}
		if cluster.Annotations == nil {
			cluster.Annotations = make(map[string]string)
		}
		cluster.Annotations[managedTagsAnnotation] = updatedTags
	})
}

func syncCreatedClusterTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster) (sets.Set[string], error) {
	tagManager := vapitags.NewManager(restSession.Client)
	categoryTags, err := tagManager.GetTagsForCategory(ctx, cluster.Spec.Cloud.VSphere.Tags.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag category %s: %w", cluster.Spec.Cloud.VSphere.Tags.CategoryID, err)
	}

	managedTags := getManagedTags(cluster)
	specTags := sets.NewString(cluster.Spec.Cloud.VSphere.Tags.Tags...)

	// 1. Ensure all tags in the spec are present in vSphere and tracked in the annotation.
	for _, tagName := range specTags.List() {
		// Check if the tag from the spec already exists in vSphere for this category.
		if tagID := filterTag(categoryTags, tagName); tagID != "" {
			// If it exists, ensure it's in our managed list.
			if !managedTags.Has(tagID) {
				managedTags.Insert(tagID)
			}
			continue
		}

		// If the tag does not exist, create it.
		tagID, err := createTag(ctx, restSession, cluster.Spec.Cloud.VSphere.Tags.CategoryID, tagName)
		if err != nil {
			return nil, fmt.Errorf("failed to create tag %s against category %s: %w", tagName, cluster.Spec.Cloud.VSphere.Tags.CategoryID, err)
		}
		managedTags.Insert(tagID)
	}

	return managedTags, nil
}

func syncDeletedClusterTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster, managedTags sets.Set[string]) (sets.Set[string], error) {
	// If there are no managed tags, there's nothing to delete.
	if managedTags.Len() == 0 {
		return managedTags, nil
	}

	tagManager := vapitags.NewManager(restSession.Client)
	// We need all tags from the category to map managed tag IDs back to tag names.
	categoryTags, err := tagManager.GetTagsForCategory(ctx, cluster.Spec.Cloud.VSphere.Tags.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag category %s: %w", cluster.Spec.Cloud.VSphere.Tags.CategoryID, err)
	}

	specTags := sets.NewString(cluster.Spec.Cloud.VSphere.Tags.Tags...)

	clusterIsDeleting := cluster.DeletionTimestamp != nil
	tagsToDelete := sets.New[string]()

	// Determine which tags to delete.
	for tagID := range managedTags {
		// Find the name of the tag corresponding to the managed ID.
		var tagName string
		for _, categoryTag := range categoryTags {
			if categoryTag.ID == tagID {
				tagName = categoryTag.Name
				break
			}
		}

		// If the tag is no longer in the spec (or the cluster is deleting), mark it for deletion.
		if !specTags.Has(tagName) || clusterIsDeleting {
			tagsToDelete.Insert(tagID)
		}
	}

	if tagsToDelete.Len() == 0 {
		return managedTags, nil
	}

	// Attempt to delete the tags from vSphere.
	for tagID := range tagsToDelete {
		tag, err := tagManager.GetTag(ctx, tagID)
		if err != nil {
			// If the tag is already gone, just remove it from our managed list.
			if isNotFound(err) {
				managedTags.Delete(tagID)
				continue
			}
			return nil, fmt.Errorf("failed to get tag %s: %w", tagID, err)
		}

		// Before deleting from vSphere, ensure it's not attached to any objects.
		attachedObjs, err := tagManager.ListAttachedObjects(ctx, tag.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list attached objects for tag %s: %w", tag.Name, err)
		}

		if len(attachedObjs) > 0 {
			// Cannot delete a tag that is still in use. We'll retry on the next reconcile.
			continue
		}

		if err := tagManager.DeleteTag(ctx, tag); err != nil {
			return nil, fmt.Errorf("failed to delete tag %s: %w", tag.Name, err)
		}

		// On successful deletion from vSphere, remove it from our managed list.
		managedTags.Delete(tagID)
	}

	return managedTags, nil
}

func getManagedTags(cluster *kubermaticv1.Cluster) sets.Set[string] {
	if cluster.Annotations == nil || cluster.Annotations[managedTagsAnnotation] == "" {
		return sets.New[string]()
	}
	return sets.New(strings.Split(cluster.Annotations[managedTagsAnnotation], ",")...)
}

func filterTag(categoryTags []vapitags.Tag, tagName string) string {
	for _, tag := range categoryTags {
		if tag.Name == tagName {
			return tag.ID
		}
	}

	return ""
}

func createTag(ctx context.Context, restSession *RESTSession, categoryID, name string) (string, error) {
	tagManager := vapitags.NewManager(restSession.Client)

	return tagManager.CreateTag(ctx, &vapitags.Tag{
		Name:       name,
		CategoryID: categoryID,
	})
}
