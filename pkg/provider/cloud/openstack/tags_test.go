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
			name:     "Multiple cluster-id tags",
			input:    []string{"cluster-id.k8c.io:id1", "cluster-id.k8c.io:id2", "cluster-id.k8c.io:id3"},
			expected: []string{"id1", "id2", "id3"},
		},
		{
			name:     "Single cluster-id tag",
			input:    []string{"cluster-id.k8c.io:id1"},
			expected: []string{"id1"},
		},
		{
			name:     "No cluster-id tags",
			input:    []string{"managed-by:kubermatic", "environment:production"},
			expected: []string{},
		},
		{
			name:     "Empty cluster-id tag",
			input:    []string{"cluster-id.k8c.io:"},
			expected: []string{},
		},
		{
			name:     "Mixed tags with one cluster-id tag",
			input:    []string{"tag1:value1", "cluster-id.k8c.io:id1", "tag2:value2"},
			expected: []string{"id1"},
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
