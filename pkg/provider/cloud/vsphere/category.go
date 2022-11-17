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

	"github.com/vmware/govmomi/vapi/tags"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func reconcileTagCategory(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) error {
	if cluster.Spec.Cloud.VSphere.TagCategory == nil {
		defaultCategory := defaultTagCategory(cluster)

		categoryID, err := fetchTagCategory(ctx, restSession, defaultCategory)
		if err != nil {
			return fmt.Errorf("failed to fetch tag category: %w", err)
		}

		if categoryID != "" {
			cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
				if !kuberneteshelper.HasFinalizer(cluster, tagCategoryCleanupFinilizer) {
					kuberneteshelper.AddFinalizer(cluster, tagCategoryCleanupFinilizer)
				}

				cluster.Spec.Cloud.VSphere.TagCategory = &kubermaticv1.TagCategory{
					TagCategoryName: defaultCategory,
					TagCategoryID:   categoryID,
				}
			})
			if err != nil {
				return fmt.Errorf("failed to add finalizer %s on vsphere cluster object: %w", tagCategoryCleanupFinilizer, err)
			}

			return nil
		}

		categoryID, err = createTagCategory(ctx, restSession, defaultCategory)
		if err != nil {
			return fmt.Errorf("failed to create default tag category: %w", err)
		}

		cluster, err = update(ctx, cluster.Name, func(cluster *kubermaticv1.Cluster) {
			if !kuberneteshelper.HasFinalizer(cluster, tagCategoryCleanupFinilizer) {
				kuberneteshelper.AddFinalizer(cluster, tagCategoryCleanupFinilizer)
			}
			cluster.Spec.Cloud.VSphere.TagCategory = &kubermaticv1.TagCategory{
				TagCategoryName: defaultCategory,
				TagCategoryID:   categoryID,
			}
		})
		if err != nil {
			return fmt.Errorf("failed to add finalizer %s on vsphere cluster object: %w", tagCategoryCleanupFinilizer, err)
		}
	}

	return nil
}

func defaultTagCategory(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("%s-cluster-%s",
		defaultCategoryPrefix,
		cluster.Name)
}

func fetchTagCategory(ctx context.Context, restSession *RESTSession, name string) (string, error) {
	tagManager := tags.NewManager(restSession.Client)
	categories, err := tagManager.GetCategories(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get tag categories %w", err)
	}

	for _, category := range categories {
		if category.Name == name {
			return category.ID, nil
		}
	}

	return "", err
}

// createTagCategory creates the specified tag category if it does not exist yet.
func createTagCategory(ctx context.Context, restSession *RESTSession, categoryName string) (string, error) {
	tagManager := tags.NewManager(restSession.Client)

	return tagManager.CreateCategory(ctx, &tags.Category{
		Name:            categoryName,
		Cardinality:     "MULTIPLE",
		AssociableTypes: []string{"Virtual Machine"},
	})
}

// deleteTagCategory deletes the tag category.
func deleteTagCategory(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster) error {
	tagManager := tags.NewManager(restSession.Client)
	categories, err := tagManager.GetCategories(ctx)
	if err != nil {
		return fmt.Errorf("failed to get tag categories %w", err)
	}

	defaultCategoryName := defaultTagCategory(cluster)

	for _, category := range categories {
		if category.Name == defaultCategoryName {
			return tagManager.DeleteCategory(ctx, &tags.Category{ID: category.ID})
		}
	}

	return nil
}
