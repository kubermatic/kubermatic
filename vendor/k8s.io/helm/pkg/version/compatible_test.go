/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

// Package version represents the current version of the project.
package version // import "k8s.io/helm/pkg/version"

import "testing"

func TestIsCompatible(t *testing.T) {
	tests := []struct {
		client   string
		server   string
		expected bool
	}{
		{"v2.0.0-alpha.4", "v2.0.0-alpha.4", true},
		{"v2.0.0-alpha.3", "v2.0.0-alpha.4", false},
		{"v2.0.0", "v2.0.0-alpha.4", false},
		{"v2.0.0-alpha.4", "v2.0.0", false},
		{"v2.0.0", "v2.0.1", true},
		{"v2.0.1", "v2.0.0", true},
		{"v2.0.0", "v2.1.1", true},
		{"v2.1.0", "v2.0.1", false},
	}

	for _, tt := range tests {
		if IsCompatible(tt.client, tt.server) != tt.expected {
			t.Errorf("expected client(%s) and server(%s) to be %v", tt.client, tt.server, tt.expected)
		}
	}
}
