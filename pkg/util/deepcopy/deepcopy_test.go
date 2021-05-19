/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package deepcopy_test

import (
	"testing"

	"github.com/go-test/deep"

	"k8c.io/kubermatic/v2/pkg/util/deepcopy"
)

func TestStringInterfaceMapCopy(t *testing.T) {

	testcases := []struct {
		name           string
		copyFrom       map[string]interface{}
		copyTo         map[string]interface{}
		expectedResult map[string]interface{}
		expectedErr    error
	}{
		{
			name: "scenario 1: deep copy one map to another",
			copyFrom: map[string]interface{}{
				"key":      []string{"values", "values"},
				"otherKey": bob{name: "Bob"},
			},
			copyTo: map[string]interface{}{},
			expectedResult: map[string]interface{}{
				"key":      []string{"values", "values"},
				"otherKey": bob{name: "Bob"},
			},
		},
		{
			name:           "scenario 2: deep copy nil map produces nil result",
			copyFrom:       nil,
			copyTo:         nil,
			expectedResult: nil,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			err := deepcopy.StringInterfaceMapCopy(tc.copyFrom, tc.copyTo)
			if err != nil {
				if tc.expectedErr != nil {

				} else {
					t.Fatal(err)
				}
			}

			if diff := deep.Equal(tc.copyFrom, tc.expectedResult); diff != nil {
				t.Errorf("Got unexpected difference in maps. Diff to expected: %v", diff)
			}

		})
	}
}

type bob struct {
	name string
}
