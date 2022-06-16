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

package yamled

import "testing"

func assertPath(t *testing.T, value Path, expected Path) {
	if len(value) != len(expected) {
		t.Fatalf("Expected path of length %d but got %d elements.", len(expected), len(value))
	}

	for idx, step := range value {
		if step != expected[idx] {
			t.Fatalf("Path should have been %v, but is %v.", expected, value)
		}
	}
}

func TestEmptyPathParent(t *testing.T) {
	assertPath(t, Path{}.Parent(), Path{})
}

func TestPathEmptyParent(t *testing.T) {
	assertPath(t, Path{"a"}.Parent(), Path{})
}

func TestPathParent(t *testing.T) {
	assertPath(t, Path{"a", "b", "c"}.Parent(), Path{"a", "b"})
}

func TestEmptyPathEnd(t *testing.T) {
	if end := (Path{}).End(); end != nil {
		t.Errorf("end of an empty path should be nil, but is %v", end)
	}
}

func TestPathEnd(t *testing.T) {
	if end := (Path{"a", "b", "c"}).End(); end != "c" {
		t.Errorf("end of [a b c] should be a, but is %v", end)
	}
}
