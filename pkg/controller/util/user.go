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

package util

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func GetUserOwnedProjects(ctx context.Context, client ctrlruntimeclient.Client, userName string) ([]kubermaticv1.Project, error) {
	var ownedProjects []kubermaticv1.Project

	projectList := &kubermaticv1.ProjectList{}
	if err := client.List(ctx, projectList); err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	for _, project := range projectList.Items {
		if len(project.OwnerReferences) == 0 {
			continue
		}

		for _, ownerReference := range project.OwnerReferences {
			ownerUser := ownerReference.Name
			if ownerUser == userName {
				ownedProjects = append(ownedProjects, project)
				break
			}
		}
	}

	return ownedProjects, nil
}
