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

package features

import (
	"testing"
)

func TestFeatureGates(t *testing.T) {
	scenarios := []struct {
		name   string
		input  string
		output map[string]bool
	}{
		{
			name:  "scenario 1: happy path provides valid input and makes sure it was parsed correctly",
			input: "feature1=false,feature2=true",
			output: map[string]bool{
				"feature1": false,
				"feature2": true,
			},
		},
	}

	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			target, err := NewFeatures(tc.input)
			if err != nil {
				t.Fatal(err)
			}
			for feature, shouldBeEnabled := range tc.output {
				isEnabled := target.Enabled(feature)
				if isEnabled != shouldBeEnabled {
					t.Fatalf("expected feature = %s to be set to %v but was set to %v", feature, shouldBeEnabled, isEnabled)
				}
			}
		})
	}
}
