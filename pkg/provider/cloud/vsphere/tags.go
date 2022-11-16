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

package vsphere

import (
	"context"
	"fmt"

	vapitags "github.com/vmware/govmomi/vapi/tags"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func reconcileTags(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	var (
		clusterTags = cluster.Spec.Cloud.VSphere.Tags
		defaultTag  = &kubermaticv1.VSphereTag{
			Name:       controllerOwnershipTag(cluster.Name),
			CategoryID: cluster.Spec.Cloud.VSphere.TagCategory.TagCategoryID,
		}
	)

	if clusterTags[defaultTag.Name] == nil {
		if clusterTags == nil {
			clusterTags = map[string]*kubermaticv1.VSphereTag{}
		}

		clusterTags[defaultTag.Name] = defaultTag
	}

	tagManager := vapitags.NewManager(restSession.Client)
	categoryTags, err := tagManager.GetTagsForCategory(ctx, cluster.Spec.Cloud.VSphere.TagCategory.TagCategoryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag %w", err)
	}

	if err := syncCreatedClusterTags(ctx, restSession, clusterTags, categoryTags, cluster.Name, update); err != nil {
		return nil, fmt.Errorf("failed to sync created tags %w", err)
	}

	if err := syncDeletedClusterTags(ctx, restSession, categoryTags, clusterTags); err != nil {
		return nil, fmt.Errorf("failed to sync deleted tags %w", err)
	}

	return cluster, nil
}

func syncCreatedClusterTags(ctx context.Context, s *RESTSession, tags map[string]*kubermaticv1.VSphereTag,
	categoryTags []vapitags.Tag, clusterName string, update provider.ClusterUpdater) error {
	for _, tag := range tags {
		fetchedTagID := filterTag(categoryTags, tag.Name)

		if fetchedTagID != "" {
			continue
		}

		tagID, err := createTag(ctx, s, tag.CategoryID, tag.Name)
		if err != nil {
			return fmt.Errorf("failed to create tag %s category: %w", tag.Name, err)
		}

		_, err = update(ctx, clusterName, func(cluster *kubermaticv1.Cluster) {
			if updatedTag, ok := cluster.Spec.Cloud.VSphere.Tags[tag.Name]; ok {
				updatedTag.ID = tagID
			}
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func syncDeletedClusterTags(ctx context.Context, s *RESTSession, categoryTags []vapitags.Tag,
	clusterTags map[string]*kubermaticv1.VSphereTag) error {
	tagManager := vapitags.NewManager(s.Client)

	for _, tag := range categoryTags {
		if _, ok := clusterTags[tag.Name]; ok {
			continue
		}

		if err := tagManager.DeleteTag(ctx, &tag); err != nil {
			return fmt.Errorf("failed to delete tag %s: %w", tag.Name, err)
		}
	}

	return nil
}

func controllerOwnershipTag(clusterName string) string {
	return fmt.Sprintf("%s-cluster-%s",
		defaultTagPrefix,
		clusterName)
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
