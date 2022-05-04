/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package email

import (
	"testing"
)

func TestMatchesRequirements(t *testing.T) {
	testcases := []struct {
		name        string
		email       string
		required    []string
		expected    bool
		expectedErr bool
	}{
		{
			name:     "no restrictions, should allow",
			email:    "foo@bar.com",
			expected: true,
		},
		{
			name:     "invalid input, but no requirements so it's fine",
			email:    "notanemail",
			expected: true,
		},
		{
			name:        "invalid input",
			email:       "notanemail",
			required:    []string{"example.com"},
			expectedErr: true,
		},
		{
			name:        "invalid requirement",
			email:       "user@example.com",
			required:    []string{"invalid@foo@bar"},
			expectedErr: true,
		},
		{
			name:     "basic check, user must be of domain",
			email:    "user@example.com",
			required: []string{"invalid.com"},
			expected: false,
		},
		{
			name:     "basic check, allow equality",
			email:    "user@example.com",
			required: []string{"user@example.com"},
			expected: true,
		},
		{
			name:     "basic check, allow localhost",
			email:    "user@localhost",
			required: []string{"localhost"},
			expected: true,
		},
		{
			name:     "ignore case differences in domain name",
			email:    "user@example.COM",
			required: []string{"example.com"},
			expected: true,
		},
		{
			name:     "do not ignore case differences in user name",
			email:    "user@example.com",
			required: []string{"USER@example.com"},
			expected: false,
		},
		{
			name:     "check all requirements",
			email:    "user@example.com",
			required: []string{"localhost", "notgoogle.com", "example.com"},
			expected: true,
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			result, err := MatchesRequirements(testcase.email, testcase.required)

			if (err != nil) != testcase.expectedErr {
				t.Fatalf("Expected error = %v, but err = %v", testcase.expectedErr, err)
			}

			if result != testcase.expected {
				t.Fatalf("Expected %v, but got %v", testcase.expected, result)
			}
		})
	}
}
