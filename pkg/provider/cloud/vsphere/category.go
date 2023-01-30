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

func reconcileTagCategory(ctx context.Context, restSession *RESTSession, cluster *kubermaticv1.Cluster, update provider.ClusterUpdater) (*kubermaticv1.Cluster, error) {
	category, err := fetchTagCategory(ctx, restSession, cluster.Spec.Cloud.VSphere.TagCategory.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch tag category: %w", err)
	}

	cluster, err = update(ctx, cluster.Name, func(updatedCluster *kubermaticv1.Cluster) {
		if !kuberneteshelper.HasFinalizer(updatedCluster, tagCleanupFinilizer) {
			kuberneteshelper.AddFinalizer(updatedCluster, tagCleanupFinilizer)
		}

		updatedCluster.Spec.Cloud.VSphere.TagCategory = &kubermaticv1.TagCategory{
			Name: category.Name,
			ID:   category.ID,
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add finalizer %s on vsphere cluster object: %w", tagCleanupFinilizer, err)
	}

	return cluster, nil
}

func fetchTagCategory(ctx context.Context, restSession *RESTSession, name string) (*tags.Category, error) {
	tagManager := tags.NewManager(restSession.Client)
	categories, err := tagManager.GetCategories(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get tag categories %w", err)
	}

	for _, category := range categories {
		if category.Name == name {
			return &category, nil
		}
	}

	return nil, err
}

func defaultClusterSpecTagCategory(ctx context.Context, spec *kubermaticv1.ClusterSpec, category *kubermaticv1.TagCategory, restSession *RESTSession) error {
	if category.ID != "" {
		tagManager := tags.NewManager(restSession.Client)
		category, err := tagManager.GetCategory(ctx, category.ID)
		if err != nil {
			return fmt.Errorf("failed to get tag categories %w", err)
		}

		spec.Cloud.VSphere.TagCategory.Name = category.Name

		return nil
	}

	if category.Name != "" {
		tagManager := tags.NewManager(restSession.Client)
		category, err := tagManager.GetCategory(ctx, category.Name)
		if err != nil {
			return fmt.Errorf("failed to get tag categories %w", err)
		}

		spec.Cloud.VSphere.TagCategory.ID = category.ID
	}

	return nil
}
