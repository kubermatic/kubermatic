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

package rbacusercluster

import (
	"fmt"
	"testing"

	"k8c.io/kubermatic/v3/pkg/controller/master-controller-manager/rbac"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/equality"
)

func TestGeneratedResourcesForGroups(t *testing.T) {
	tests := []struct {
		name              string
		resourceName      string
		expectError       bool
		expectedResources []string
	}{
		{
			name:              "scenario 1: check resources for owners",
			resourceName:      genResourceName(rbac.OwnerGroupNamePrefix),
			expectedResources: []string{"*"},
			expectError:       false,
		},
		{
			name:              "scenario 2: check resources for editors",
			resourceName:      genResourceName(rbac.EditorGroupNamePrefix),
			expectedResources: []string{"*"},
			expectError:       false,
		},
		{
			name:              "scenario 3: check resources for viewers",
			resourceName:      genResourceName(rbac.ViewerGroupNamePrefix),
			expectedResources: []string{"applicationinstallations", "machinedeployments", "machinesets", "machines"},
			expectError:       false,
		},
		{
			name:         "scenario 4: incorrect resource name",
			resourceName: rbac.ViewerGroupNamePrefix,
			expectError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr, err := CreateClusterRole(test.resourceName, &rbacv1.ClusterRole{})
			if test.expectError && err == nil {
				t.Fatal("expected error")
			}
			if !test.expectError && err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}
			if test.expectError {
				return
			}

			// check across all rules if the expected resource exists
			for _, resource := range test.expectedResources {
				found := false
				for _, rule := range cr.Rules {
					if contains(rule.Resources, resource) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected resource %q was not found in rulegroup %+v\n", resource, cr.Rules)
				}
			}
		})
	}
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func TestGenerateVerbsForGroup(t *testing.T) {
	tests := []struct {
		name          string
		resourceName  string
		expectError   bool
		expectedVerbs []string
	}{
		// test for any named resource
		{
			name:          "scenario 1: generate verbs for owners",
			resourceName:  genResourceName(rbac.OwnerGroupNamePrefix),
			expectedVerbs: []string{"*"},
			expectError:   false,
		},
		{
			name:          "scenario 2: generate verbs for editors",
			resourceName:  genResourceName(rbac.EditorGroupNamePrefix),
			expectedVerbs: []string{"*"},
			expectError:   false,
		},
		{
			name:          "scenario 3: generate verbs for viewers",
			resourceName:  genResourceName(rbac.ViewerGroupNamePrefix),
			expectedVerbs: []string{"list", "get", "watch"},
			expectError:   false,
		},
		{
			name:         "scenario 4: incorrect resource name",
			resourceName: rbac.ViewerGroupNamePrefix,
			expectError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cr, err := CreateClusterRole(test.resourceName, &rbacv1.ClusterRole{})
			if test.expectError && err == nil {
				t.Fatal("expected error")
			}
			if !test.expectError && err != nil {
				t.Fatalf("expected no error, but got: %v", err)
			}
			if test.expectError {
				return
			}

			returnedVerbs := cr.Rules[0].Verbs
			if !equality.Semantic.DeepEqual(returnedVerbs, test.expectedVerbs) {
				t.Fatalf("incorrect verbs were returned, got: %v, want: %v", returnedVerbs, test.expectedVerbs)
			}
		})
	}
}

func TestGroupName(t *testing.T) {
	tests := []struct {
		name              string
		resourceName      string
		expectError       bool
		expectedGroupName string
	}{
		{
			name:              "scenario 1: get group name for owners",
			resourceName:      genResourceName(rbac.OwnerGroupNamePrefix),
			expectError:       false,
			expectedGroupName: rbac.OwnerGroupNamePrefix,
		},
		{
			name:              "scenario 2: get group name for viewers",
			resourceName:      genResourceName(rbac.ViewerGroupNamePrefix),
			expectError:       false,
			expectedGroupName: rbac.ViewerGroupNamePrefix,
		},
		{
			name:              "scenario 3: get group name for editors",
			resourceName:      genResourceName(rbac.EditorGroupNamePrefix),
			expectError:       false,
			expectedGroupName: rbac.EditorGroupNamePrefix,
		},
		{
			name:         "scenario 4: incorrect resource name",
			resourceName: "test:test:test",
			expectError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			groupName, err := getGroupName(test.resourceName)

			if test.expectError {
				if err == nil {
					t.Fatalf("expected error")
				}
			} else {
				if err != nil {
					t.Fatalf("getting group name from resource name failed with error: %v", err)
				}

				if groupName != test.expectedGroupName {
					t.Fatalf("incorrect group name was returned, got %q, want %q", groupName, test.expectedGroupName)
				}
			}
		})
	}
}

func genResourceName(groupName string) string {
	return fmt.Sprintf("system:%s:%s", rbac.RBACResourcesNamePrefix, groupName)
}
