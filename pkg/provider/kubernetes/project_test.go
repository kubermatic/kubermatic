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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	testProject = "test"
)

func TestProjectGetterFactory(t *testing.T) {
	testCases := []struct {
		name               string
		expectedProjectMap map[string]*kubermaticv1.Project
		seedClient         ctrlruntimeclient.Client
	}{
		{
			name: "scenario 1: should return one existing project",
			expectedProjectMap: map[string]*kubermaticv1.Project{
				testProject: genProject(testProject),
			},
			seedClient: fake.
				NewClientBuilder().
				WithObjects(genProject(testProject)).
				Build(),
		},
		{
			name:               "scenario 2: should return empty map when no project exists",
			expectedProjectMap: map[string]*kubermaticv1.Project{},
			seedClient: fake.
				NewClientBuilder().
				WithObjects().
				Build(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			projectsGetter, err := ProjectsGetterFactory(context.Background(), tc.seedClient)
			if err != nil {
				t.Fatalf("failed getting projectsGetter: %v", err)
			}
			projects, err := projectsGetter()
			if err != nil {
				t.Fatalf("failed calling projectsGetter: %v", err)
			}

			if cmp := diff.ObjectDiff(tc.expectedProjectMap, projects); cmp != "" {
				t.Fatalf("expected projects map is not equal to current one: %s", cmp)
			}
		})
	}
}

func genProject(name string) *kubermaticv1.Project {
	project := &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			ResourceVersion: "1",
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: name,
		},
	}
	return project
}
