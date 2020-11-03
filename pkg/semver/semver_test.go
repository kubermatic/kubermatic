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

package semver

import (
	"bytes"
	"testing"
)

func TestMarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		inputSemver    *Semver
		expectedResult []byte
	}{
		{
			name:           "simple semver struct",
			inputSemver:    NewSemverOrDie("v1.0.0"),
			expectedResult: []byte("\"1.0.0\""),
		},
		{
			name:           "simple semver struct 2",
			inputSemver:    NewSemverOrDie("v2.1.0"),
			expectedResult: []byte("\"2.1.0\""),
		},
		{
			name:           "simple semver struct 3",
			inputSemver:    NewSemverOrDie("v3.2.1"),
			expectedResult: []byte("\"3.2.1\""),
		},
		{
			name:           "no-v semver",
			inputSemver:    NewSemverOrDie("4.3.2"),
			expectedResult: []byte("\"4.3.2\""),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, err := tc.inputSemver.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(tc.expectedResult, b) {
				t.Errorf("expected to get %s, but got %s", string(tc.expectedResult), string(b))
			}

			var s Semver
			if err = s.UnmarshalJSON(b); err != nil {
				t.Fatal(err)
			}
			if !s.Equal(tc.inputSemver) {
				t.Errorf("expected to get %s, but got %s", tc.inputSemver.String(), s.String())
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		inputByte      []byte
		expectedSemver *Semver
	}{
		{
			name:           "simple semver struct",
			inputByte:      []byte("\"1.0.0\""),
			expectedSemver: NewSemverOrDie("v1.0.0"),
		},
		{
			name:           "simple semver struct 2",
			inputByte:      []byte("\"2.1.0\""),
			expectedSemver: NewSemverOrDie("v2.1.0"),
		},
		{
			name:           "simple semver struct 3",
			inputByte:      []byte("\"3.2.1\""),
			expectedSemver: NewSemverOrDie("v3.2.1"),
		},
		{
			name:           "no-v semver",
			inputByte:      []byte("\"4.3.2\""),
			expectedSemver: NewSemverOrDie("4.3.2"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var s Semver
			if err := s.UnmarshalJSON(tc.inputByte); err != nil {
				t.Fatal(err)
			}
			if !s.Equal(tc.expectedSemver) {
				t.Errorf("expected to get %s, but got %s", tc.expectedSemver.String(), s.String())
			}

			b, err := s.MarshalJSON()
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(b, tc.inputByte) {
				t.Errorf("expected to get %s, but got %s", string(tc.inputByte), string(b))
			}
		})
	}
}
