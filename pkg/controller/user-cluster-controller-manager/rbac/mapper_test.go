package rbacusercluster

import (
	"fmt"
	"testing"

	"github.com/kubermatic/kubermatic/pkg/controller/master-controller-manager/rbac"

	"k8s.io/apimachinery/pkg/api/equality"
)

func TestGeneratedResourcesForGroups(t *testing.T) {
	tests := []struct {
		name              string
		resurceName       string
		expectError       bool
		expectedResources []string
	}{
		{
			name:              "scenario 1: check resources for owners",
			resurceName:       genResourceName(rbac.OwnerGroupNamePrefix),
			expectedResources: []string{"*"},
			expectError:       false,
		},
		{
			name:              "scenario 2: check resources for editors",
			resurceName:       genResourceName(rbac.EditorGroupNamePrefix),
			expectedResources: []string{"*"},
			expectError:       false,
		},
		{
			name:              "scenario 3: check resources for viewers",
			resurceName:       genResourceName(rbac.ViewerGroupNamePrefix),
			expectedResources: []string{"machinedeployments", "machinesets", "machines"},
			expectError:       false,
		},
		{
			name:        "scenario 4: incorrect resource name",
			resurceName: rbac.ViewerGroupNamePrefix,
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			role, err := GenerateRBACClusterRole(test.resurceName)

			if test.expectError {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("generate RBAC role err: %v", err)
			}

			actualResources := role.Rules[0].Resources
			if !equality.Semantic.DeepEqual(actualResources, test.expectedResources) {
				t.Fatalf("incorrect resources were returned, got: %v, want: %v", actualResources, test.expectedResources)
			}

		})
	}
}

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
			resurceName:   genResourceName(rbac.OwnerGroupNamePrefix),
			expectedVerbs: []string{"*"},
			expectError:   false,
		},
		{
			name:          "scenario 2: generate verbs for editors",
			resurceName:   genResourceName(rbac.EditorGroupNamePrefix),
			expectedVerbs: []string{"*"},
			expectError:   false,
		},
		{
			name:          "scenario 3: generate verbs for viewers",
			resurceName:   genResourceName(rbac.ViewerGroupNamePrefix),
			expectedVerbs: []string{"list", "get", "watch"},
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
			role, err := GenerateRBACClusterRole(test.resurceName)

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

func TestGroupName(t *testing.T) {

	tests := []struct {
		name              string
		resurceName       string
		expectError       bool
		expectedGroupName string
	}{
		{
			name:              "scenario 1: get group name for owners",
			resurceName:       genResourceName(rbac.OwnerGroupNamePrefix),
			expectError:       false,
			expectedGroupName: rbac.OwnerGroupNamePrefix,
		},
		{
			name:              "scenario 2: get group name for viewers",
			resurceName:       genResourceName(rbac.ViewerGroupNamePrefix),
			expectError:       false,
			expectedGroupName: rbac.ViewerGroupNamePrefix,
		},
		{
			name:              "scenario 3: get group name for editors",
			resurceName:       genResourceName(rbac.EditorGroupNamePrefix),
			expectError:       false,
			expectedGroupName: rbac.EditorGroupNamePrefix,
		},
		{
			name:        "scenario 4: incorrect resource name",
			resurceName: "test:test:test",
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			groupName, err := getGroupName(test.resurceName)

			if test.expectError {
				if err == nil {
					t.Fatalf("expected error")
				}

			} else {
				if err != nil {
					t.Fatalf("getting group name from resource name failed with error: %v", err)
				}

				if groupName != test.expectedGroupName {
					t.Fatalf("incorrect group name was returned, got: %v, want: %v", groupName, test.expectedGroupName)
				}
			}

		})
	}
}

func genResourceName(groupName string) string {
	return fmt.Sprintf("system:%s:%s", rbac.RBACResourcesNamePrefix, groupName)
}
