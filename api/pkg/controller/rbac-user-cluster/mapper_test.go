package rbacusercluster

import (
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestGenerateVerbsForGroup(t *testing.T) {

	tests := []struct {
		name          string
		groupName     string
		expectedVerbs []string
	}{
		// test for any named resource
		{
			name:          "scenario 1: generate verbs for owners",
			groupName:     rbac.OwnerGroupNamePrefix,
			expectedVerbs: []string{"create", "list", "get", "update", "delete"},
		},
		{
			name:          "scenario 2: generate verbs for editors",
			groupName:     rbac.EditorGroupNamePrefix,
			expectedVerbs: []string{"create", "list", "get", "update", "delete"},
		},
		{
			name:          "scenario 3: generate verbs for viewers",
			groupName:     rbac.ViewerGroupNamePrefix,
			expectedVerbs: []string{"list", "get"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			role, err := GenerateRBACClusterRole(test.groupName)
			if err != nil {
				t.Fatalf("generate RBAC role err: %v", err)
			}

			returnedVerbs := role.Rules[0].Verbs
			if !equality.Semantic.DeepEqual(returnedVerbs, test.expectedVerbs) {
				t.Fatalf("incorrect verbs were returned, got: %v, want: %v", returnedVerbs, test.expectedVerbs)
			}

		})
	}
}
