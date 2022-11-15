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
)

func tagCategoryOwnership(cluster *kubermaticv1.Cluster) string {
	return fmt.Sprintf("%s/cluster-%s-%s",
		defaultCategoryPrefix,
		cluster.Name, cluster.Spec.Cloud.VSphere.TagCategoryName)
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
func createTagCategory(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster) (string, error) {
	tagManager := tags.NewManager(restSession.Client)

	defaultCategoryName := tagCategoryOwnership(cluster)

	return tagManager.CreateCategory(ctx, &tags.Category{
		Name:            defaultCategoryName,
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

	defaultCategoryName := tagCategoryOwnership(cluster)

	for _, category := range categories {
		if category.Name == defaultCategoryName {
			return tagManager.DeleteCategory(ctx, &tags.Category{ID: category.ID})
		}
	}

	return nil
}
