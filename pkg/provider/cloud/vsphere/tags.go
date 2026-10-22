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

	vapitags "github.com/vmware/govmomi/vapi/tags"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/util/sets"
)

// reconcileTags makes sure all tags from the cluster spec exist and removes the tags that KKP manages
// for this cluster but that are not in the spec anymore. It returns the set of tag IDs that are
// managed by KKP for this cluster; the caller is responsible for recording them in the cluster status.
func reconcileTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, sets.Set[string], error) {
	managedTags, err := syncCreatedClusterTags(ctx, restSession, cluster)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sync created tags: %w", err)
	}

	managedTags, err = syncDeletedClusterTags(ctx, restSession, cluster, managedTags)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sync deleted tags: %w", err)
	}

	cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		if !kuberneteshelper.HasFinalizer(cluster, tagCleanupFinalizer) {
			kuberneteshelper.AddFinalizer(cluster, tagCleanupFinalizer)
		}
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to add finalizer %s on vsphere cluster object: %w", tagCleanupFinalizer, err)
	}

	return cluster, managedTags, nil
}

// syncCreatedClusterTags creates all tags from the cluster spec that do not exist yet and returns
// the tags KKP manages for this cluster. Tags from the spec that already exist (for example because a
// previous reconciliation was interrupted before the status could be persisted, or because the cluster
// was created before KKP tracked tag ownership) are adopted.
func syncCreatedClusterTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster) (sets.Set[string], error) {
	categoryID := cluster.Spec.Cloud.VSphere.Tags.CategoryID

	tagManager := vapitags.NewManager(restSession.Client)
	categoryTags, err := tagManager.GetTagsForCategory(ctx, categoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag category %s: %w", categoryID, err)
	}

	managedTags := getManagedTags(cluster)

	for _, tagName := range cluster.Spec.Cloud.VSphere.Tags.Tags {
		tagID := filterTag(categoryTags, tagName)
		if tagID == "" {
			tagID, err = createTag(ctx, restSession, categoryID, tagName)
			if err != nil {
				return nil, fmt.Errorf("failed to create tag %s against category %s: %w", tagName, categoryID, err)
			}
		}

		managedTags.Insert(tagID)
	}

	return managedTags, nil
}

// syncDeletedClusterTags deletes managed tags that are not part of the cluster spec anymore, or all
// of them if the cluster is being deleted. Tags that are not managed by this cluster are never touched.
// It returns the tags that are still managed afterwards.
func syncDeletedClusterTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster, managedTags sets.Set[string]) (sets.Set[string], error) {
	if managedTags.Len() == 0 {
		return managedTags, nil
	}

	tagManager := vapitags.NewManager(restSession.Client)
	categoryTags, err := tagManager.GetTagsForCategory(ctx, cluster.Spec.Cloud.VSphere.Tags.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag category %s: %w", cluster.Spec.Cloud.VSphere.Tags.CategoryID, err)
	}

	specTags := sets.New(cluster.Spec.Cloud.VSphere.Tags.Tags...)
	clusterIsDeleting := cluster.DeletionTimestamp != nil
	remaining := managedTags.Clone()

	for tagID := range managedTags {
		var tag *vapitags.Tag
		for i := range categoryTags {
			if categoryTags[i].ID == tagID {
				tag = &categoryTags[i]
				break
			}
		}

		// The tag does not exist anymore (e.g. removed manually), nothing left to manage.
		if tag == nil {
			remaining.Delete(tagID)
			continue
		}

		if specTags.Has(tag.Name) && !clusterIsDeleting {
			continue
		}

		attachedObjs, err := tagManager.ListAttachedObjects(ctx, tag.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to list attached objects for the given tag %s: %w", tag.Name, err)
		}

		// if there are still objects attached to the tag, we can't delete it; it stays
		// managed so that we retry on the next reconciliation.
		if len(attachedObjs) > 0 {
			continue
		}

		if err := tagManager.DeleteTag(ctx, tag); err != nil {
			return nil, fmt.Errorf("failed to delete tag %s: %w", tag.Name, err)
		}

		remaining.Delete(tagID)
	}

	return remaining, nil
}

// cleanupClusterTags deletes the tags managed for a cluster that is being deleted. Clusters that were
// created before KKP tracked tag ownership have no recorded tags, so the tags from their spec that
// exist in the category are treated as managed as well.
func cleanupClusterTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster) error {
	tagManager := vapitags.NewManager(restSession.Client)
	categoryTags, err := tagManager.GetTagsForCategory(ctx, cluster.Spec.Cloud.VSphere.Tags.CategoryID)
	if err != nil {
		return fmt.Errorf("failed to get tag category %s: %w", cluster.Spec.Cloud.VSphere.Tags.CategoryID, err)
	}

	managedTags := getManagedTags(cluster)
	for _, tagName := range cluster.Spec.Cloud.VSphere.Tags.Tags {
		if tagID := filterTag(categoryTags, tagName); tagID != "" {
			managedTags.Insert(tagID)
		}
	}

	_, err = syncDeletedClusterTags(ctx, restSession, cluster, managedTags)

	return err
}

// getManagedTags returns the IDs of the tags KKP manages for this cluster in the tag category from the spec.
func getManagedTags(cluster *kubermaticv1.Cluster) sets.Set[string] {
	if cluster.Status.CloudProvider == nil || cluster.Status.CloudProvider.VSphere == nil {
		return sets.New[string]()
	}

	return sets.New(cluster.Status.CloudProvider.VSphere.ManagedTags[cluster.Spec.Cloud.VSphere.Tags.CategoryID]...)
}

// setManagedTags records the tags KKP manages for this cluster in the in-memory cluster status.
// Persisting the status is up to the caller.
func setManagedTags(cluster *kubermaticv1.Cluster, managedTags sets.Set[string]) {
	categoryID := cluster.Spec.Cloud.VSphere.Tags.CategoryID

	if cluster.Status.CloudProvider == nil {
		cluster.Status.CloudProvider = &kubermaticv1.CloudProviderStatus{}
	}
	if cluster.Status.CloudProvider.VSphere == nil {
		cluster.Status.CloudProvider.VSphere = &kubermaticv1.VSphereCloudProviderStatus{}
	}

	status := cluster.Status.CloudProvider.VSphere
	if managedTags.Len() == 0 {
		delete(status.ManagedTags, categoryID)
		return
	}

	if status.ManagedTags == nil {
		status.ManagedTags = map[string][]string{}
	}
	status.ManagedTags[categoryID] = sets.List(managedTags)
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
