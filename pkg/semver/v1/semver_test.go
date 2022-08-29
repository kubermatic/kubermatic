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

package v1

import "testing"

func TestDeepCopy(t *testing.T) {
	tt := []struct {
		name string
		in   *Semver
		exp  string
	}{
		{
			name: "full version prefixed by v",
			in:   NewSemverOrDie("v1.0.0"),
			exp:  "1.0.0",
		},
		{
			name: "partial version prefixed by v",
			in:   NewSemverOrDie("v1"),
			exp:  "1.0.0",
		},
		{
			name: "full version no prefix",
			in:   NewSemverOrDie("1.0.0"),
			exp:  "1.0.0",
		},
		{
			name: "partial version no prefix",
			in:   NewSemverOrDie("1"),
			exp:  "1.0.0",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			cp := tc.in.DeepCopy()

			// we need to test for string equality here, as both semver.String() as well as
			// NewSemverOrDie take a biased stance on transforming the version
			if string(cp) != tc.exp {
				t.Errorf("Expected copy to be %q, got %q", string(*tc.in), string(cp))
			}
		})
	}
}
