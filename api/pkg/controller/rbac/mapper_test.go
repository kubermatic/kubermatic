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
			name:          "scenario 3: editors of a project cannot delete the project",
			groupName:     "editors-projectID",
			expectedVerbs: []string{"get", "update"},
			resourceKind:  "Project",
		},
		{
			name:          "scenario 4: viewers of a project can view any named resource",
			groupName:     "viewers-projectID",
			expectedVerbs: []string{"get"},
			resourceKind:  "",
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
		expectedVerbs []string
	}{
		{
			name:          "scenario 1: owners of a project can create resources for the given project",
			groupName:     "owners-projectID",
			expectedVerbs: []string{"create"},
		},
		{
			name:          "scenario 2: editors of a project can create resources for the given project",
			groupName:     "editors-projectID",
			expectedVerbs: []string{"create"},
		},
		{
			name:          "scenario 3: viewers of a project cannot create any resources for the given project",
			groupName:     "viewers-projectID",
			expectedVerbs: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if returnedVerbs, err := generateVerbsForResource(test.groupName); err != nil || !equality.Semantic.DeepEqual(returnedVerbs, test.expectedVerbs) {
				t.Fatalf("incorrect verbs were returned, got: %v, want: %v, err: %v", returnedVerbs, test.expectedVerbs, err)
			}
		})
	}
}
