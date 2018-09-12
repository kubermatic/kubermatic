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
			groupName:     "owners-projectID",
			expectedVerbs: []string{"get", "update", "delete"},
			resourceKind:  "",
		},
		{
			name:          "scenario 2: editors of a project can read, update and delete almost any named resource",
			groupName:     "editors-projectID",
			expectedVerbs: []string{"get", "update", "delete"},
			resourceKind:  "",
		},
		{
			name:          "scenario 3: viewers of a project can view any named resource",
			groupName:     "viewers-projectID",
			expectedVerbs: []string{"get"},
			resourceKind:  "",
		},

		// test for Project named resource
		{
			name:          "scenario 4: editors of a project cannot delete the project",
			groupName:     "editors-projectID",
			expectedVerbs: []string{"get", "update"},
			resourceKind:  "Project",
		},

		// tests for UserProjectBinding named resource
		{
			name:          "scenario 5: owners of a project can interact with UserProjectBinding named resource",
			groupName:     "owners-projectID",
			expectedVerbs: []string{"get", "update", "delete"},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 6: editors of a project cannot interact with UserProjectBinding named resource",
			groupName:     "editors-projectID",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 7: viewers of a project cannot interact with UserProjectBinding named resource",
			groupName:     "viewers-projectID",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
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
		groupName     string
		resourceKind  string
		expectedVerbs []string
	}{
		{
			name:          "scenario 1: owners of a project can create project resources",
			groupName:     "owners-projectID",
			expectedVerbs: []string{"create"},
			resourceKind:  "Project",
		},
		{
			name:          "scenario 2: editors of a project can create project resources",
			groupName:     "editors-projectID",
			expectedVerbs: []string{"create"},
			resourceKind:  "Project",
		},
		{
			name:          "scenario 3: viewers of a project cannot create any resources for the given project",
			groupName:     "viewers-projectID",
			resourceKind:  "Project",
			expectedVerbs: []string{},
		},
		{
			name:          "scenario 4: owners of a project can create any resource that is considered project's resource",
			groupName:     "owners-projectID",
			expectedVerbs: []string{"create"},
		},
		{
			name:          "scenario 5: editors of a project can create any resource that is considered project's resource",
			groupName:     "editors-projectID",
			expectedVerbs: []string{"create"},
		},
		{
			name:          "scenario 6: owners of a project can create UserProjectBinding resource",
			groupName:     "owners-projectID",
			expectedVerbs: []string{"create"},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 7: editors of a project cannot create UserProjectBinding resource",
			groupName:     "editors-projectID",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 7: viewers of a project cannot create UserProjectBinding resource",
			groupName:     "viewers-projectID",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if returnedVerbs, err := generateVerbsForResource(test.groupName, test.resourceKind); err != nil || !equality.Semantic.DeepEqual(returnedVerbs, test.expectedVerbs) {
				t.Fatalf("incorrect verbs were returned, got: %v, want: %v, err: %v", returnedVerbs, test.expectedVerbs, err)
			}
		})
	}
}
