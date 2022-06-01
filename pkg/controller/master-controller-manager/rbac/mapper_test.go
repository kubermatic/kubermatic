/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package rbac

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestGenerateVerbsForNamedResources(t *testing.T) {
	tests := []struct {
		name          string
		groupName     string
		resourceKind  string
		expectedVerbs []string
	}{
		// test for any named resource
		{
			name:          "scenario 1: owners of a project can read, update and delete any named resource",
			groupName:     "owners",
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "",
		},
		{
			name:          "scenario 2: editors of a project can read, update and delete almost any named resource",
			groupName:     "editors",
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "",
		},
		{
			name:          "scenario 3: viewers of a project can view any named resource",
			groupName:     "viewers",
			expectedVerbs: []string{"get"},
			resourceKind:  "",
		},
		{
			name:          "scenario 4: projectmanagers of a project can manage any named resource",
			groupName:     "projectmanagers",
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "",
		},

		// test for Project named resource
		{
			name:          "scenario 5: editors of a project cannot delete the project",
			groupName:     "editors",
			expectedVerbs: []string{"get", "update", "patch"},
			resourceKind:  "Project",
		},

		// tests for UserProjectBinding named resource
		{
			name:          "scenario 6: owners of a project can interact with UserProjectBinding named resource",
			groupName:     "owners",
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 7: editors of a project cannot interact with UserProjectBinding named resource",
			groupName:     "editors",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 8: viewers of a project cannot interact with UserProjectBinding named resource",
			groupName:     "viewers",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 9: viewers of a project cannot interact with ServiceAccount (User) named resource",
			groupName:     "viewers",
			expectedVerbs: []string{},
			resourceKind:  "User",
		},
		{
			name:          "scenario 10: editors of a project cannot interact with ServiceAccount (User) named resource",
			groupName:     "editors",
			expectedVerbs: []string{},
			resourceKind:  "User",
		},
		{
			name:          "scenario 11: projectmanagers of a project can interact with ServiceAccount (User) named resource",
			groupName:     "projectmanagers",
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "User",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if returnedVerbs, err := generateVerbsForNamedResource(test.groupName, test.resourceKind); err != nil || !equality.Semantic.DeepEqual(returnedVerbs, test.expectedVerbs) {
				t.Fatalf("incorrect verbs were returned, got: %v, want: %v, err: %v", returnedVerbs, test.expectedVerbs, err)
			}
		})
	}
}

func TestGenerateVerbsForResources(t *testing.T) {
	tests := []struct {
		name          string
		roleName      string
		resourceKind  string
		expectedVerbs []string
	}{
		{
			name:          "scenario 1: owners of a project can create project resources",
			roleName:      "owners",
			expectedVerbs: []string{"create"},
			resourceKind:  "Project",
		},
		{
			name:          "scenario 2: editors of a project can create project resources",
			roleName:      "editors",
			expectedVerbs: []string{"create"},
			resourceKind:  "Project",
		},
		{
			name:          "scenario 3: viewers of a project cannot create any resources for the given project",
			roleName:      "viewers",
			resourceKind:  "Project",
			expectedVerbs: []string{},
		},
		{
			name:          "scenario 4: owners of a project can create any resource that is considered project's resource",
			roleName:      "owners",
			expectedVerbs: []string{"create"},
		},
		{
			name:          "scenario 5: editors of a project can create any resource that is considered project's resource",
			roleName:      "editors",
			expectedVerbs: []string{"create"},
		},
		{
			name:          "scenario 6: owners of a project can create UserProjectBinding resource",
			roleName:      "owners",
			expectedVerbs: []string{"create"},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 7: editors of a project cannot create UserProjectBinding resource",
			roleName:      "editors",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 7: viewers of a project cannot create UserProjectBinding resource",
			roleName:      "viewers",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 8: only the owners can create ServiceAccounts (aka. User) resources",
			roleName:      "owners",
			expectedVerbs: []string{"create"},
			resourceKind:  "User",
		},
		{
			name:          "scenario 9: the editors cannot create ServiceAccounts (aka. User) resources",
			roleName:      "editors",
			expectedVerbs: []string{},
			resourceKind:  "User",
		},
		{
			name:          "scenario 10: the viewers cannot create ServiceAccounts (aka. User) resources",
			roleName:      "viewers",
			expectedVerbs: []string{},
			resourceKind:  "User",
		},
		{
			name:          "scenario 11: the projectmanagers can create ServiceAccounts (aka. User) resources",
			roleName:      "projectmanagers",
			expectedVerbs: []string{"create"},
			resourceKind:  "User",
		},
		{
			name:          "scenario 12: the projectmanagers can create UserProjectBinding resources",
			roleName:      "projectmanagers",
			expectedVerbs: []string{"create"},
			resourceKind:  "UserProjectBinding",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if returnedVerbs, err := generateVerbsForResource(test.roleName, test.resourceKind); err != nil || !equality.Semantic.DeepEqual(returnedVerbs, test.expectedVerbs) {
				t.Fatalf("incorrect verbs were returned, got: %v, want: %v, err: %v", returnedVerbs, test.expectedVerbs, err)
			}
		})
	}
}
