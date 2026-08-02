/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestGetParsedValues(t *testing.T) {
	tt := map[string]struct {
		appIn       ApplicationInstallationSpec
		expResponse map[string]interface{}
		expError    bool
	}{
		"Values set": {
			appIn: ApplicationInstallationSpec{
				Values: runtime.RawExtension{Raw: []byte(`{"not-empty":"value"}`)},
			},
			expResponse: map[string]interface{}{"not-empty": "value"},
			expError:    false,
		},
		"ValuesBlock set": {
			appIn: ApplicationInstallationSpec{
				ValuesBlock: "not-empty:\n  value",
			},
			expResponse: map[string]interface{}{"not-empty": "value"},
			expError:    false,
		},
		"ValuesBlock set and Values Defaulted": {
			appIn: ApplicationInstallationSpec{
				Values:      runtime.RawExtension{Raw: []byte("{}")},
				ValuesBlock: "not-empty:\n  value",
			},
			expResponse: map[string]interface{}{"not-empty": "value"},
			expError:    false,
		},
		"Both Values and ValuesBlock set": {
			appIn: ApplicationInstallationSpec{
				Values:      runtime.RawExtension{Raw: []byte(`{"not-empty":"value"}`)},
				ValuesBlock: "not-empty:\n  value",
			},
			expResponse: nil,
			expError:    true,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			res, err := tc.appIn.GetParsedValues()
			if tc.expError == false && err != nil {
				t.Errorf("Expected Error to be nil, but got %v", err)
			}
			if tc.expError == true && err == nil {
				t.Errorf("Expected Error to be present, but was nil")
			}
			if diff := cmp.Diff(tc.expResponse, res); diff != "" {
				t.Fatalf("Got unexpected response:\n%s", diff)
			}
		})
	}
}
