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

package openstack

import (
	"fmt"
	"reflect"
	"testing"
)

func TestOwnersFromTags(t *testing.T) {
	testCases := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "Single owned-by-cluster tag with multiple IDs",
			input:    []string{"managed-by:kubermatic", "owned-by-cluster:id1,id2,id3"},
			expected: []string{"id1", "id2", "id3"},
		},
		{
			name:     "Single owned-by-cluster tag with one ID",
			input:    []string{"owned-by-cluster:id1"},
			expected: []string{"id1"},
		},
		{
			name:     "No owned-by-cluster tag",
			input:    []string{"managed-by:kubermatic", "environment:production"},
			expected: []string{},
		},
		{
			name:     "Empty owned-by-cluster tag",
			input:    []string{"owned-by-cluster:"},
			expected: []string{},
		},
		{
			name:     "Trailing comma in owned-by-cluster tag",
			input:    []string{"owned-by-cluster:id1,id2,"},
			expected: []string{"id1", "id2"},
		},
		{
			name:     "Multiple commas in owned-by-cluster tag",
			input:    []string{"owned-by-cluster:id1,,id2"},
			expected: []string{"id1", "id2"},
		},
		{
			name:     "Mixed tags with no owned-by-cluster",
			input:    []string{"tag1:value1", "tag2:value2"},
			expected: []string{},
		},
		{
			name:     "Empty input tags slice",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ownersFromTags(tc.input)
			if !reflect.DeepEqual(result, tc.expected) {
				fmt.Println(len(result), len(tc.expected))
				t.Errorf("\nTest %q failed\nGot:    %v\nWanted: %v", tc.name, result, tc.expected)
			}
		})
	}
}

func TestAddTagOwnership(t *testing.T) {
	testCases := []struct {
		name      string
		input     []string
		clusterID string
		expected  []string
	}{
		{
			name:      "Add cluster to existing owned-by-cluster tag",
			input:     []string{"managed-by:kubermatic", "owned-by-cluster:id1,id2"},
			clusterID: "id3",
			expected:  []string{"managed-by:kubermatic", "owned-by-cluster:id1,id2,id3"},
		},
		{
			name:      "Add cluster to empty owned-by-cluster tag",
			input:     []string{"owned-by-cluster:"},
			clusterID: "id1",
			expected:  []string{"owned-by-cluster:id1"},
		},
		{
			name:      "Add duplicate cluster to owned-by-cluster tag",
			input:     []string{"owned-by-cluster:id1,id2"},
			clusterID: "id2",
			expected:  []string{"owned-by-cluster:id1,id2"},
		},
		{
			name:      "No owned-by-cluster tag",
			input:     []string{"managed-by:kubermatic", "environment:production"},
			clusterID: "id1",
			expected:  []string{"managed-by:kubermatic", "environment:production"},
		},
		{
			name:      "Handle trailing commas in owned-by-cluster tag",
			input:     []string{"owned-by-cluster:id1,id2,"},
			clusterID: "id3",
			expected:  []string{"owned-by-cluster:id1,id2,id3"},
		},
		{
			name:      "Handle multiple commas in owned-by-cluster tag",
			input:     []string{"owned-by-cluster:id1,,id2"},
			clusterID: "id3",
			expected:  []string{"owned-by-cluster:id1,id2,id3"},
		},
		{
			name:      "Empty input tags slice",
			input:     []string{},
			clusterID: "id1",
			expected:  []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := addTagOwnership(tc.input, tc.clusterID)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Test %q failed:\nGot:    %v\nWanted: %v", tc.name, result, tc.expected)
			}
		})
	}
}

// DeleteClusterFromRouterTags removes a cluster name from the "owned-by-cluster" tag in the tags slice.
func TestRemoveTagOwnership(t *testing.T) {
	testCases := []struct {
		name      string
		input     []string
		clusterID string
		expected  []string
	}{
		{
			name:      "Delete cluster from existing owned-by-cluster tag",
			input:     []string{"managed-by:kubermatic", "owned-by-cluster:id1,id2,id3"},
			clusterID: "id2",
			expected:  []string{"managed-by:kubermatic", "owned-by-cluster:id1,id3"},
		},
		{
			name:      "Delete only cluster from owned-by-cluster tag",
			input:     []string{"owned-by-cluster:id1"},
			clusterID: "id1",
			expected:  []string{},
		},
		{
			name:      "Delete non-existent cluster from owned-by-cluster tag",
			input:     []string{"owned-by-cluster:id1,id2"},
			clusterID: "id3",
			expected:  []string{"owned-by-cluster:id1,id2"},
		},
		{
			name:      "No owned-by-cluster tag",
			input:     []string{"managed-by:kubermatic", "environment:production"},
			clusterID: "id1",
			expected:  []string{"managed-by:kubermatic", "environment:production"},
		},
		{
			name:      "Handle trailing commas in owned-by-cluster tag",
			input:     []string{"owned-by-cluster:id1,id2,"},
			clusterID: "id1",
			expected:  []string{"owned-by-cluster:id2"},
		},
		{
			name:      "Handle multiple commas in owned-by-cluster tag",
			input:     []string{"owned-by-cluster:id1,,id2"},
			clusterID: "id2",
			expected:  []string{"owned-by-cluster:id1"},
		},
		{
			name:      "Empty input tags slice",
			input:     []string{},
			clusterID: "id1",
			expected:  []string{},
		},
		{
			name:      "Malformed owned-by-cluster tag with only commas",
			input:     []string{"owned-by-cluster:,,,"},
			clusterID: "id1",
			expected:  []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := removeTagOwnership(tc.input, tc.clusterID)
			if !reflect.DeepEqual(result, tc.expected) {
				t.Errorf("Test %q failed:\nGot:    %v\nWanted: %v", tc.name, result, tc.expected)
			}
		})
	}
}
