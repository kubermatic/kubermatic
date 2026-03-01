/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
)

func TestParseProviderFilter(t *testing.T) {
	testCases := []struct {
		name          string
		input         []string
		expectedCount int
		expectError   bool
	}{
		{
			name:          "empty filter returns nil",
			input:         []string{},
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "nil filter returns nil",
			input:         nil,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "single valid provider",
			input:         []string{"aws"},
			expectedCount: 1,
			expectError:   false,
		},
		{
			name:          "multiple valid providers",
			input:         []string{"aws", "azure", "kubevirt"},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "mixed case providers",
			input:         []string{"AWS", "Azure", "KubeVirt"},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "providers with spaces",
			input:         []string{" aws ", " azure ", " kubevirt "},
			expectedCount: 3,
			expectError:   false,
		},
		{
			name:          "invalid provider",
			input:         []string{"aws", "invalid-provider", "azure"},
			expectedCount: 0,
			expectError:   true,
		},
		{
			name:          "duplicate providers",
			input:         []string{"aws", "azure", "aws"},
			expectedCount: 2,
			expectError:   false,
		},
		{
			name:          "empty strings in slice",
			input:         []string{"aws", "", "azure"},
			expectedCount: 2,
			expectError:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := parseProviderFilter(tc.input)

			if tc.expectError {
				if err == nil {
					t.Errorf("expected an error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tc.expectedCount == 0 && result != nil {
				t.Errorf("expected nil result for empty filter, got %v", result)
				return
			}

			if tc.expectedCount > 0 {
				if result == nil {
					t.Errorf("expected non-nil result, got nil")
					return
				}

				if result.Len() != tc.expectedCount {
					t.Errorf("expected %d providers, got %d: %v", tc.expectedCount, result.Len(), sets.List(result))
				}
			}
		})
	}
}
