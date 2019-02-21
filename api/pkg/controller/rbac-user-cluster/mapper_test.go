package rbacusercluster

import (
	"fmt"
	"testing"

	"github.com/kubermatic/kubermatic/api/pkg/controller/rbac"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestGenerateVerbsForGroup(t *testing.T) {

	tests := []struct {
		name          string
		resurceName   string
		expectError   bool
		expectedVerbs []string
	}{
		// test for any named resource
		{
			name:          "scenario 1: generate verbs for owners",
			resurceName:   fmt.Sprintf("system:kubermatic:%s", rbac.OwnerGroupNamePrefix),
			expectedVerbs: []string{"create", "list", "get", "update", "delete"},
			expectError:   false,
		},
		{
			name:          "scenario 2: generate verbs for editors",
			resurceName:   fmt.Sprintf("system:kubermatic:%s", rbac.EditorGroupNamePrefix),
			expectedVerbs: []string{"create", "list", "get", "update", "delete"},
			expectError:   false,
		},
		{
			name:          "scenario 3: generate verbs for viewers",
			resurceName:   fmt.Sprintf("system:kubermatic:%s", rbac.ViewerGroupNamePrefix),
			expectedVerbs: []string{"list", "get"},
			expectError:   false,
		},
		{
			name:        "scenario 4: incorrect resource name",
			resurceName: rbac.ViewerGroupNamePrefix,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			role, err := generateRBACClusterRole(test.resurceName)

			if test.expectError {
				if err == nil {
					t.Fatalf("expected error")
				}

			} else {
				if err != nil {
					t.Fatalf("generate RBAC role err: %v", err)
				}

				returnedVerbs := role.Rules[0].Verbs
				if !equality.Semantic.DeepEqual(returnedVerbs, test.expectedVerbs) {
					t.Fatalf("incorrect verbs were returned, got: %v, want: %v", returnedVerbs, test.expectedVerbs)
				}
			}

		})
	}
}
