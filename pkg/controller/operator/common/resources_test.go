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

import (
	"fmt"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"
)

func TestComparableVersionSuffix(t *testing.T) {
	testcases := []struct {
		greater string
		smaller string
	}{
		// { X > Y }
		{"v1.10", "v1.9"},
		{"v1.0.1", "v1.0.0"},
		{"v1.0.10", "v1.0.9"},
		{"v1.0.0", "v1.0.0-beta.1"},
		{"v1.0.0-alpha.1", "v1.0.0-alpha.0"},
		{"v1.0.0-beta.0", "v1.0.0-alpha.2"},
		{"v1.0.0-1-gabcdef", "v1.0.0"},
		{"v1.0.0-10-gabcdef", "v1.0.0-9-gabcdef"},
		{"v1.0.1", "v1.0.0-9-gabcdef"},
		{"v1.0.1-beta.1-9-gabcdef", "v1.0.0-beta.1"},
		{"v1.0.1-beta.1-10-gabcdef", "v1.0.0-beta.1-9-gabcdef"},
	}

	for _, testcase := range testcases {
		t.Run(fmt.Sprintf("%s > %s", testcase.greater, testcase.smaller), func(t *testing.T) {
			smaller, err := semverlib.NewVersion(comparableVersionSuffix(testcase.smaller))
			if err != nil {
				t.Fatalf("Failed to parse smaller value %q: %v", testcase.smaller, err)
			}

			greater, err := semverlib.NewVersion(comparableVersionSuffix(testcase.greater))
			if err != nil {
				t.Fatalf("Failed to parse greater value %q: %v", testcase.greater, err)
			}

			if !greater.GreaterThan(smaller) {
				t.Fatalf("Comparing %q > %q after patching (%q > %q) should have yielded true.", testcase.greater, testcase.smaller, greater.String(), smaller.String())
			}
		})
	}
}
