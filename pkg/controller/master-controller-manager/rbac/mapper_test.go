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
	"fmt"
	"reflect"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

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
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "",
		},
		{
			name:          "scenario 2: editors of a project can read, update and delete almost any named resource",
			groupName:     "editors-projectID",
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "",
		},
		{
			name:          "scenario 3: viewers of a project can view any named resource",
			groupName:     "viewers-projectID",
			expectedVerbs: []string{"get"},
			resourceKind:  "",
		},
		{
			name:          "scenario 4: projectmanagers of a project can manage any named resource",
			groupName:     "projectmanagers-projectID",
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "",
		},

		// test for Project named resource
		{
			name:          "scenario 5: editors of a project cannot delete the project",
			groupName:     "editors-projectID",
			expectedVerbs: []string{"get", "update", "patch"},
			resourceKind:  "Project",
		},

		// tests for UserProjectBinding named resource
		{
			name:          "scenario 6: owners of a project can interact with UserProjectBinding named resource",
			groupName:     "owners-projectID",
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 7: editors of a project cannot interact with UserProjectBinding named resource",
			groupName:     "editors-projectID",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 8: viewers of a project cannot interact with UserProjectBinding named resource",
			groupName:     "viewers-projectID",
			expectedVerbs: []string{},
			resourceKind:  "UserProjectBinding",
		},
		{
			name:          "scenario 9: viewers of a project cannot interact with ServiceAccount (User) named resource",
			groupName:     "viewers-projectID",
			expectedVerbs: []string{},
			resourceKind:  "User",
		},
		{
			name:          "scenario 10: editors of a project cannot interact with ServiceAccount (User) named resource",
			groupName:     "editors-projectID",
			expectedVerbs: []string{},
			resourceKind:  "User",
		},
		{
			name:          "scenario 11: projectmanagers of a project can interact with ServiceAccount (User) named resource",
			groupName:     "projectmanagers-projectID",
			expectedVerbs: []string{"get", "update", "patch", "delete"},
			resourceKind:  "User",
		},
		{
			name:          "scenario 12: the owners can get ResourceQuota resource",
			groupName:     "owners-projectID",
			expectedVerbs: []string{"get"},
			resourceKind:  "ResourceQuota",
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
		{
			name:          "scenario 8: only the owners can create ServiceAccounts (aka. User) resources",
			groupName:     "owners-projectID",
			expectedVerbs: []string{"create"},
			resourceKind:  "User",
		},
		{
			name:          "scenario 9: the editors cannot create ServiceAccounts (aka. User) resources",
			groupName:     "editors-projectID",
			expectedVerbs: []string{},
			resourceKind:  "User",
		},
		{
			name:          "scenario 10: the viewers cannot create ServiceAccounts (aka. User) resources",
			groupName:     "viewers-projectID",
			expectedVerbs: []string{},
			resourceKind:  "User",
		},
		{
			name:          "scenario 11: the projectmanagers can create ServiceAccounts (aka. User) resources",
			groupName:     "projectmanagers-projectID",
			expectedVerbs: []string{"create"},
			resourceKind:  "User",
		},
		{
			name:          "scenario 12: the projectmanagers can create UserProjectBinding resources",
			groupName:     "projectmanagers-projectID",
			expectedVerbs: []string{"create"},
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

func TestIsAcceptedNamespace(t *testing.T) {
	tests := []struct {
		namespace string
		expected  bool
	}{
		{resources.KubermaticNamespace, true},
		{fmt.Sprintf("%s_namespace", resources.KubeOneNamespacePrefix), true},
		{"different_namespace", false},
		{"kubermatic_suffix", false}, // Test for suffix condition
		{"", false},                  // Test for empty string
	}

	for _, tc := range tests {
		actual := isAcceptedNamespace(tc.namespace)
		if actual != tc.expected {
			t.Errorf("check(%q) = %v, expected %v", tc.namespace, actual, tc.expected)
		}
	}
}

func TestGenerateVerbsForNamespacedResource(t *testing.T) {
	tests := []struct {
		name          string
		groupName     string
		resourceKind  string
		namespace     string
		expectedVerbs []string
		wantErr       bool
	}{
		// Test successful creation of secrets in project namespaces
		{
			name:          "Owner in project namespace (secret)",
			groupName:     OwnerGroupNamePrefix + "-2wsx3edc",
			resourceKind:  secretV1Kind,
			namespace:     resources.KubeOneNamespacePrefix + "my-project",
			expectedVerbs: []string{"create"},
			wantErr:       false,
		},
		{
			name:          "Project manager in project namespace (secret)",
			groupName:     ProjectManagerGroupNamePrefix + "-2wsx3edc",
			resourceKind:  secretV1Kind,
			namespace:     resources.KubeOneNamespacePrefix + "your-project",
			expectedVerbs: []string{"create"},
			wantErr:       false,
		},
		{
			name:          "Non-owner in project namespace (secret)",
			groupName:     "other-group",
			resourceKind:  secretV1Kind,
			namespace:     resources.KubeOneNamespacePrefix + "other-project",
			expectedVerbs: nil,
			wantErr:       false,
		},
		// Test successful creation of CBSL in kubermatic namespace
		{
			name:          "Owner in kubermatic namespace (CBSL)",
			groupName:     OwnerGroupNamePrefix + "-2wsx3edc",
			resourceKind:  kubermaticv1.ClusterBackupStorageLocationKind,
			namespace:     resources.KubermaticNamespace,
			expectedVerbs: []string{"create"},
			wantErr:       false,
		},
		{
			name:          "Editor in kubermatic namespace (CBSL)",
			groupName:     EditorGroupNamePrefix + "-2wsx3edc",
			resourceKind:  kubermaticv1.ClusterBackupStorageLocationKind,
			namespace:     resources.KubermaticNamespace,
			expectedVerbs: []string{"create"},
			wantErr:       false,
		},
		{
			name:          "Project manager in kubermatic namespace (CBSL)",
			groupName:     ProjectManagerGroupNamePrefix + "-2wsx3edc",
			resourceKind:  kubermaticv1.ClusterBackupStorageLocationKind,
			namespace:     resources.KubermaticNamespace,
			expectedVerbs: []string{"create"},
			wantErr:       false,
		},
		// Test denied creation of CBSL in non-kubermatic namespace
		{
			name:          "Owner in non-kubermatic namespace (CBSL)",
			groupName:     OwnerGroupNamePrefix + "-2wsx3edc",
			resourceKind:  kubermaticv1.ClusterBackupStorageLocationKind,
			namespace:     "other-namespace",
			expectedVerbs: nil,
			wantErr:       true,
		},
		// Test unknown group
		{
			name:          "Unknown group",
			groupName:     "unknown-group",
			resourceKind:  "unknown-kind",
			namespace:     resources.KubermaticNamespace,
			expectedVerbs: nil,
			wantErr:       true,
		},
	}

	for _, tc := range tests {
		actualVerbs, actualError := generateVerbsForNamespacedResource(tc.groupName, tc.resourceKind, tc.namespace)
		if !reflect.DeepEqual(actualVerbs, tc.expectedVerbs) || (tc.wantErr && actualError == nil) || (!tc.wantErr && actualError != nil) {
			t.Errorf("Test: %s - generateVerbsForNamespacedResource(%q, %q, %q) = %v, %v; expected %v, wantErr: %v", tc.name, tc.groupName, tc.resourceKind, tc.namespace, actualVerbs, actualError, tc.expectedVerbs, tc.wantErr)
		}
	}
}

func TestGenerateVerbsForNamedResourceInNamespace(t *testing.T) {
	tests := []struct {
		name          string
		groupName     string
		resourceKind  string
		namespace     string
		expectedVerbs []string
		wantErr       bool
	}{
		// Namespace and kind acceptance cases
		{
			name:          "Owner can get, update, and delete secrets in saSecretsNamespaceName",
			groupName:     OwnerGroupNamePrefix + "-group",
			resourceKind:  secretV1Kind,
			namespace:     saSecretsNamespaceName,
			expectedVerbs: []string{"get", "update", "delete"},
			wantErr:       false,
		},
		{
			name:          "Project Manager can get, update, and delete secrets in KubeOneNamespacePrefix namespace",
			groupName:     ProjectManagerGroupNamePrefix + "-group",
			resourceKind:  secretV1Kind,
			namespace:     resources.KubeOneNamespacePrefix + "some-namespace",
			expectedVerbs: []string{"get", "update", "delete"},
			wantErr:       false,
		},
		{
			name:          "Unsupported Namespace for secrets",
			groupName:     OwnerGroupNamePrefix + "-group",
			resourceKind:  secretV1Kind,
			namespace:     "unsupported-namespace",
			expectedVerbs: nil,
			wantErr:       true,
		},
		{
			name:          "Unsupported Kind for secrets",
			groupName:     OwnerGroupNamePrefix + "-group",
			resourceKind:  "unsupported-kind",
			namespace:     saSecretsNamespaceName,
			expectedVerbs: nil,
			wantErr:       true,
		},

		// CBSL in KubermaticNamespace
		{
			name:          "Owner can fully manage CBSL in KubermaticNamespace",
			groupName:     OwnerGroupNamePrefix + "-group",
			resourceKind:  kubermaticv1.ClusterBackupStorageLocationKind,
			namespace:     resources.KubermaticNamespace,
			expectedVerbs: []string{"get", "list", "create", "patch", "update", "delete"},
			wantErr:       false,
		},
		// Unknown Group
		{
			name:          "Unknown Group",
			groupName:     "unknown-group",
			resourceKind:  secretV1Kind,
			namespace:     resources.KubermaticNamespace,
			expectedVerbs: nil,
			wantErr:       false,
		},
	}

	for _, tc := range tests {
		actualVerbs, actualError := generateVerbsForNamedResourceInNamespace(tc.groupName, tc.resourceKind, tc.namespace)
		if !reflect.DeepEqual(actualVerbs, tc.expectedVerbs) || (tc.wantErr && actualError == nil) || (!tc.wantErr && actualError != nil) {
			t.Errorf("Test: %s - generateVerbsForNamedResourceInNamespace(%q, %q, %q) = %v, %v; expected %v, wantErr: %v", tc.name, tc.groupName, tc.resourceKind, tc.namespace, actualVerbs, actualError, tc.expectedVerbs, tc.wantErr)
		}
	}
}
