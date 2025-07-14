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

package kubernetes

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8c.io/kubermatic/v2/pkg/provider"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// emptyProjectMap is returned when no project is present.
	emptyProjectMap = map[string]*kubermaticv1.Project{}
)

func ProjectsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (provider.ProjectsGetter, error) {
	return func() (map[string]*kubermaticv1.Project, error) {
		projectList := &kubermaticv1.ProjectList{}
		if err := client.List(ctx, projectList); err != nil {
			if apierrors.IsNotFound(err) {
				// We should not fail if no project exists and just return an
				// empty map.
				return emptyProjectMap, nil
			}
			return nil, fmt.Errorf("failed to get project %q: %w", provider.DefaultSeedName, err)
		}
		projectMap := map[string]*kubermaticv1.Project{}
		for _, project := range projectList.Items {
			projectMap[project.Name] = &project
		}
		return projectMap, nil
	}, nil
}
