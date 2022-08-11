/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package common

import "testing"

func TestComparableVersionSuffix(t *testing.T) {
	testcases := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "",
		},
		{
			input:    "v1.0",
			expected: "v1.0",
		},
		{
			input:    "v1.0.1",
			expected: "v1.0.1",
		},
		{
			input:    "v1.0.1-beta",
			expected: "v1.0.1-beta",
		},
		{
			input:    "v1.0.1-beta.1",
			expected: "v1.0.1-beta.1",
		},
		{
			input:    "v1.0.1-beta.1-randomsuffix",
			expected: "v1.0.1-beta.1-randomsuffix",
		},
		{
			input:    "v1.0.1-beta.1-1-gabcdef",
			expected: "v1.0.1-beta.1-000000001",
		},
		{
			input:    "v1.0.1-beta.1-123-gabcdef",
			expected: "v1.0.1-beta.1-000000123",
		},
		{
			input:    "v1.0.1-123-gabcdef",
			expected: "v1.0.1-000000123",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.input, func(t *testing.T) {
			output := comparableVersionSuffix(testcase.input)
			if output != testcase.expected {
				t.Fatalf("Expected %q, got %q.", testcase.expected, output)
			}
		})
	}
}
