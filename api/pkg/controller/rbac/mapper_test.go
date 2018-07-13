package rbac

import (
	"reflect"
	"testing"
)

func TestGenerateVerbs(t *testing.T) {

	tests := []struct {
		groupName     string
		resourceKind  string
		expectedVerbs []string
	}{
		{
			groupName:     "owners-group",
			expectedVerbs: []string{"create", "get", "update", "delete"},
			resourceKind:  "",
		},
		{
			groupName:     "editors-group",
			expectedVerbs: []string{"create", "get", "update", "delete"},
			resourceKind:  "",
		},
		{
			groupName:     "editors-group",
			expectedVerbs: []string{"create", "get", "update"},
			resourceKind:  "Project",
		},
		{
			groupName:     "viewers-group",
			expectedVerbs: []string{"get"},
			resourceKind:  "",
		},
	}

	for _, test := range tests {
		if returnedVerbs, err := generateVerbs(test.groupName, test.resourceKind); err != nil || !reflect.DeepEqual(returnedVerbs, test.expectedVerbs) {
			t.Fatalf("Failed (%v) %v, got: %v, want: %v", err, test.groupName, returnedVerbs, test.expectedVerbs)
		}
	}
}
