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

package registry

import (
	"testing"
)

func TestImageRewriter(t *testing.T) {
	addDigest := func(s string) string {
		return s + "@sha256:0b2f19895de281e4a416700b17a4dc9b8d3b80eb7b5b65dac173880f5113084e"
	}

	testcases := []struct {
		name      string
		overwrite string
		input     string
		expected  string
	}{
		{
			name:      "default registry is applied",
			overwrite: "",
			input:     "foo/bar",
			expected:  "docker.io/foo/bar",
		},
		{
			name:      "tags are kept",
			overwrite: "",
			input:     "foo/bar:v1.2.3",
			expected:  "docker.io/foo/bar:v1.2.3",
		},
		{
			name:      "digests are kept if the registry is unchanged (with default registry)",
			overwrite: "",
			input:     addDigest("foo/bar:v1.2.3"),
			expected:  addDigest("docker.io/foo/bar:v1.2.3"),
		},
		{
			name:      "digests are kept if the registry is unchanged",
			overwrite: "",
			input:     addDigest("docker.io/foo/bar:v1.2.3"),
			expected:  addDigest("docker.io/foo/bar:v1.2.3"),
		},
		{
			name:      "untagged digests are kept if the registry is unchanged (with default registry)",
			overwrite: "",
			input:     addDigest("foo/bar"),
			expected:  addDigest("docker.io/foo/bar"),
		},
		{
			name:      "untagged digests are kept if the registry is unchanged",
			overwrite: "",
			input:     addDigest("docker.io/foo/bar"),
			expected:  addDigest("docker.io/foo/bar"),
		},
		{
			name:      "images can survive unchanged",
			overwrite: "",
			input:     "docker.io/foo/bar",
			expected:  "docker.io/foo/bar",
		},
		{
			name:      "a registry overwrite is applied",
			overwrite: "registry.local",
			input:     "docker.io/foo/bar",
			expected:  "registry.local/foo/bar",
		},
		{
			name:      "a registry overwrite will not remove the digest",
			overwrite: "registry.local",
			input:     addDigest("docker.io/foo/bar:v1.2.3"),
			expected:  addDigest("registry.local/foo/bar:v1.2.3"),
		},
		{
			name:      "a NOP rewrite should keep the digest",
			overwrite: "registry.local",
			input:     addDigest("registry.local/foo/bar:v1.2.3"),
			expected:  addDigest("registry.local/foo/bar:v1.2.3"),
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.name, func(t *testing.T) {
			rewriter := GetImageRewriterFunc(testcase.overwrite)

			output, err := rewriter(testcase.input)
			if err != nil {
				t.Fatalf("Expected no error, but got: %v", err)
			}

			if output != testcase.expected {
				t.Fatalf("Expected %q to write to %q (with overwrite %q), but got %q.", testcase.input, testcase.expected, testcase.overwrite, output)
			}
		})
	}
}
