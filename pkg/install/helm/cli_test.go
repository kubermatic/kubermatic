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

package helm

import (
	"testing"

	semverlib "github.com/Masterminds/semver/v3"
)

func TestGuessReleaseVersion(t *testing.T) {
	testcases := []struct {
		input           string
		expectedVersion *semverlib.Version
		expectedChart   string
	}{
		{
			input:           "foo",
			expectedVersion: nil,
			expectedChart:   "",
		},
		{
			input:           "foo-bar",
			expectedVersion: nil,
			expectedChart:   "",
		},
		{
			input:           "foo-1.2.3",
			expectedVersion: semverlib.MustParse("1.2.3"),
			expectedChart:   "foo",
		},
		{
			input:           "foo-bar-1.2.3",
			expectedVersion: semverlib.MustParse("1.2.3"),
			expectedChart:   "foo-bar",
		},
		{
			input:           "foo-bar-super-long-release-name-1.2.3",
			expectedVersion: semverlib.MustParse("1.2.3"),
			expectedChart:   "foo-bar-super-long-release-name",
		},
		{
			input:           "foo-bar-super-long-release-name-1.2.3-suffix-really-long",
			expectedVersion: semverlib.MustParse("1.2.3-suffix-really-long"),
			expectedChart:   "foo-bar-super-long-release-name",
		},
		{
			input:           "this-is-not-a-version",
			expectedVersion: nil,
			expectedChart:   "",
		},
	}

	for _, testcase := range testcases {
		t.Run(testcase.input, func(t *testing.T) {
			version, chart, err := guessChartName(testcase.input)
			if testcase.expectedVersion == nil && err == nil {
				t.Fatalf("Expected an error, but got version %v and chart %q.", version, chart)
			}
			if testcase.expectedVersion != nil {
				if !version.Equal(testcase.expectedVersion) {
					t.Fatalf("Expected version %v, but got version %v.", testcase.expectedVersion, version)
				}

				if testcase.expectedChart != chart {
					t.Fatalf("Expected chart %q, but got chart %q.", testcase.expectedChart, chart)
				}
			}
		})
	}
}
