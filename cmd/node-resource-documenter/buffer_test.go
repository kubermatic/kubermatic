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

package main

import (
	"strings"
	"testing"
)

func TestBuffer(t *testing.T) {
	tests := []struct {
		name     string
		inputA   []string
		inputB   []string
		expected string
	}{
		{
			name:     "empty input",
			expected: "",
		}, {
			name: "single line",
			inputA: []string{
				"one single line",
			},
			expected: "one single line",
		}, {
			name: "multiple lines",
			inputA: []string{
				"multiple",
				"\n",
				"lines",
			},
			expected: "multiple\nlines",
		}, {
			name: "add to empty",
			inputB: []string{
				"added ",
				"content",
			},
			expected: "added content",
		}, {
			name: "add to filled buffer",
			inputA: []string{
				"input a",
				"\n",
			},
			inputB: []string{
				"and",
				"\n",
				"input b",
			},
			expected: "input a\nand\ninput b",
		},
	}

	for i, test := range tests {
		t.Logf("#%d: %s", i, test.name)
		ba := newBuffer()
		ba.push(test.inputA...)
		if len(test.inputB) > 0 {
			bb := newBuffer()
			bb.push(test.inputB...)
			ba.pushAll(bb)
		}
		var builder strings.Builder
		err := ba.writeAll(&builder)
		if err != nil {
			t.Errorf("writing failed: %v", err)
		}
		if builder.String() != test.expected {
			t.Errorf("buffer content doesn't match expected: %q <> %q", builder.String(), test.expected)
		}
	}
}
