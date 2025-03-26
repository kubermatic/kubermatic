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

func reconcileTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	if err := syncCreatedClusterTags(ctx, restSession, cluster); err != nil {
		return nil, fmt.Errorf("failed to sync created tags %w", err)
	}

	if err := syncDeletedClusterTags(ctx, restSession, cluster); err != nil {
		return nil, fmt.Errorf("failed to sync deleted tags %w", err)
	}

	cluster, err := update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
		if !kuberneteshelper.HasFinalizer(cluster, tagCleanupFinalizer) {
			kuberneteshelper.AddFinalizer(cluster, tagCleanupFinalizer)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add finalizer %s on vsphere cluster object: %w", tagCleanupFinalizer, err)
	}

	return cluster, nil
}

func syncCreatedClusterTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster) error {
	tagManager := vapitags.NewManager(restSession.Client)
	categoryTags, err := tagManager.GetTagsForCategory(ctx, cluster.Spec.Cloud.VSphere.Tags.CategoryID)
	if err != nil {
		return fmt.Errorf("failed to get tag category %s: %w", cluster.Spec.Cloud.VSphere.Tags.CategoryID, err)
	}

	for _, vsphereTag := range cluster.Spec.Cloud.VSphere.Tags.Tags {
		if filterTag(categoryTags, vsphereTag) == "" {
			_, err := createTag(ctx, restSession, cluster.Spec.Cloud.VSphere.Tags.CategoryID, vsphereTag)
			if err != nil {
				return fmt.Errorf("failed to create tag %s against category %s: %w", vsphereTag, cluster.Spec.Cloud.VSphere.Tags.CategoryID, err)
			}
		}
	}

	return nil
}

func syncDeletedClusterTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster) error {
	tagManager := vapitags.NewManager(restSession.Client)
	categoryTags, err := tagManager.GetTagsForCategory(ctx, cluster.Spec.Cloud.VSphere.Tags.CategoryID)
	if err != nil {
		return fmt.Errorf("failed to get tag %w", err)
	}

	clusterTags := sets.NewString(cluster.Spec.Cloud.VSphere.Tags.Tags...)
	for _, vsphereTag := range categoryTags {
		if _, ok := clusterTags[vsphereTag.Name]; ok {
			if cluster.DeletionTimestamp == nil {
				continue
			}
		}

		// Fetch all objects attached to the tag.
		attachedObjs, err := tagManager.ListAttachedObjects(ctx, vsphereTag.ID)
		if err != nil {
			return fmt.Errorf("failed to list attached objects for the given tag %s: %w", vsphereTag.Name, err)
		}

		// if there are still objects attached to the tag, we can't delete it.
		if len(attachedObjs) > 0 {
			continue
		}

		if err := tagManager.DeleteTag(ctx, &vsphereTag); err != nil {
			return fmt.Errorf("failed to delete tag %s: %w", vsphereTag.Name, err)
		}
	}

	return nil
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
